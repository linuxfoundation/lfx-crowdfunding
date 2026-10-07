// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

// UserLookup resolves a users row by LF SSO username (UserRepository.GetByUsername).
type UserLookup func(ctx context.Context, username string) (*models.User, error)

// RequireSubjectBinding rejects an Auth0 caller whose verified sub differs from
// the legacy_user_id already bound to the users row their username claim
// resolves to. Without it, a username claim equal to a migrated row's stripped
// subject (e.g. the bare GitHub id of a legacy github|N user) would resolve to
// that legacy account. Must run after Middleware.
//
// Heimdall-issued tokens are exempt: their subject is the plain username, so
// there is no Auth0 sub to compare. Rows without a legacy_user_id (and callers
// with no row yet) pass through.
func RequireSubjectBinding(lookup UserLookup, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p := PrincipalFromContext(r.Context())
			if p == nil {
				jsonError(w, http.StatusUnauthorized, "authentication required")
				return
			}
			if p.IsHeimdallIssued {
				next.ServeHTTP(w, r)
				return
			}
			u, err := lookup(r.Context(), p.Username)
			switch {
			case errors.Is(err, domain.ErrUserNotFound):
				// First login: no row yet, nothing to hijack. SyncProfile creates it.
			case err != nil:
				logger.ErrorContext(r.Context(), "auth: subject binding lookup failed", "error", err)
				jsonError(w, http.StatusServiceUnavailable, "upstream unavailable")
				return
			case u.LegacyUserID != "" && u.LegacyUserID != p.UserID:
				logger.WarnContext(r.Context(), "auth: token sub does not match the legacy_user_id bound to this username",
					"username", p.Username, "path", r.URL.Path)
				jsonError(w, http.StatusForbidden, "identity mismatch")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
