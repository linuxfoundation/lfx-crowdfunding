// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/auth"
)

// UserHandler holds Chi handlers for the /v1/me resource.
type UserHandler struct {
	userRepo        domain.UserRepository
	userInfoFetcher auth.UserInfoFetcher
}

// NewUserHandler creates a UserHandler.
func NewUserHandler(userRepo domain.UserRepository, fetcher auth.UserInfoFetcher) *UserHandler {
	return &UserHandler{userRepo: userRepo, userInfoFetcher: fetcher}
}

// SyncProfile handles PATCH /v1/me.
//
// Called immediately after a successful login to persist the user's full
// profile in the users table. Profile data (name, email, picture) is fetched
// from the Auth0 UserInfo endpoint using the incoming access token, so that
// access tokens themselves need only carry username and email claims (REQ-P1).
func (h *UserHandler) SyncProfile(w http.ResponseWriter, r *http.Request) {
	principal := auth.PrincipalFromContext(r.Context())
	if principal == nil || principal.Username == "" {
		Error(w, domain.ErrUnauthorized)
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

	JSON(w, http.StatusOK, result)
}
