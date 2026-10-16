// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/auth"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/clients"
)

// UserMetadataLookup reads a user's profile by username (auth-service over NATS).
type UserMetadataLookup interface {
	UserMetadata(ctx context.Context, username string) (*clients.UserMetadata, error)
}

// UserHandler holds Chi handlers for the /crowdfunding/me resource.
type UserHandler struct {
	userRepo        domain.UserRepository
	userInfoFetcher auth.UserInfoFetcher
	metadataLookup  UserMetadataLookup // nil when FGA_NATS_URL is unset
	logger          *slog.Logger
}

// NewUserHandler creates a UserHandler. lookup may be nil.
func NewUserHandler(userRepo domain.UserRepository, fetcher auth.UserInfoFetcher, lookup UserMetadataLookup, logger *slog.Logger) *UserHandler {
	return &UserHandler{userRepo: userRepo, userInfoFetcher: fetcher, metadataLookup: lookup, logger: logger}
}

// SyncProfile handles PATCH /crowdfunding/me.
//
// Called immediately after a successful login to persist the user's full
// profile in the users table. Profile data (name, email, picture) is normally
// fetched from the Auth0 UserInfo endpoint using the incoming access token, so
// that access tokens themselves need only carry username and email claims
// (REQ-P1).
//
// Heimdall-issued tokens (LFXV2-3351) are minted by Heimdall's create_jwt
// finalizer and replace the original Auth0 access token before it reaches this
// service, so Auth0 rejects them if forwarded to UserInfo. For those, skip the
// UserInfo call: take the email from the validated token (Heimdall only mints
// it when verified) and the name/avatar from auth-service, best-effort (#387).
// Upsert preserves any previously-synced profile field the sync leaves empty
// instead of blanking it out. The local mock-bypass
// principal takes the same path: its UserID is the bare username, not an Auth0
// sub, so it must not be bound as a legacy_user_id.
func (h *UserHandler) SyncProfile(w http.ResponseWriter, r *http.Request) {
	principal := auth.PrincipalFromContext(r.Context())
	if principal == nil || principal.Username == "" {
		Error(w, domain.ErrUnauthorized)
		return
	}

	if principal.IsHeimdallIssued || principal.IsMockBypass {
		result, err := h.userRepo.Upsert(r.Context(), h.heimdallProfile(r.Context(), principal))
		if err != nil {
			Error(w, err)
			return
		}
		JSON(w, http.StatusOK, result)
		return
	}

	// Extract the raw access token to call Auth0 UserInfo.
	// Use case-insensitive prefix check (RFC 6750 § 2.1) to match the jwt middleware.
	authParts := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
	if len(authParts) != 2 || !strings.EqualFold(authParts[0], "bearer") {
		Error(w, domain.ErrUnauthorized)
		return
	}
	rawToken := strings.TrimSpace(authParts[1])
	if rawToken == "" {
		Error(w, domain.ErrUnauthorized)
		return
	}

	info, err := h.userInfoFetcher.FetchUserInfo(r.Context(), rawToken)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrUserInfoTokenRejected):
			// Token was rejected by Auth0 UserInfo (4xx) — treat as 401 so the
			// client knows to re-authenticate rather than retrying the request.
			Error(w, domain.ErrUnauthorized)
		case errors.Is(err, auth.ErrUserInfoUnavailable):
			// Network error or Auth0 5xx — surface as 503 so callers can retry.
			Error(w, domain.ErrUpstreamUnavailable)
		default:
			Error(w, err)
		}
		return
	}

	// Cross-check the UserInfo sub against the token principal to defend against
	// token substitution: Auth0 always returns a sub matching the token's sub.
	// An empty sub in the UserInfo response is also rejected — a valid UserInfo
	// response always contains a non-empty sub claim.
	if info.Sub == "" || info.Sub != principal.UserID {
		Error(w, domain.ErrUnauthorized)
		return
	}

	user := &models.User{
		Username:     principal.Username,
		LegacyUserID: info.Sub, // persist Auth0 sub for legacy FK lookups
		Email:        info.EffectiveEmail(),
		Name:         info.Name,
		GivenName:    info.GivenName,
		FamilyName:   info.FamilyName,
		AvatarURL:    info.Picture,
	}

	result, err := h.userRepo.Upsert(r.Context(), user)
	if err != nil {
		Error(w, err)
		return
	}

	// Upsert keeps an already-bound legacy_user_id; a different one means this
	// username's row belongs to another Auth0 identity (e.g. a migrated legacy
	// account), so never hand it to the caller.
	if result.LegacyUserID != principal.UserID {
		Error(w, domain.ErrForbidden)
		return
	}

	JSON(w, http.StatusOK, result)
}

// heimdallProfile builds the user for a Heimdall or mock-bypass principal. A
// failed auth-service lookup only costs the name fields; it never fails the sync.
func (h *UserHandler) heimdallProfile(ctx context.Context, principal *models.Principal) *models.User {
	user := &models.User{Username: principal.Username, Email: principal.Email}
	if h.metadataLookup == nil {
		return user
	}
	md, err := h.metadataLookup.UserMetadata(ctx, principal.Username)
	if err != nil {
		h.logger.WarnContext(ctx, "auth-service profile lookup failed; syncing without name",
			"username", principal.Username, "error", err)
		return user
	}
	user.Name = md.Name
	user.GivenName = md.GivenName
	user.FamilyName = md.FamilyName
	user.AvatarURL = md.Picture
	return user
}
