// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package auth provides JWT validation for Auth0-issued tokens.
package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

// DefaultClockSkew is the default leeway applied when validating JWT expiry.
const DefaultClockSkew = 5 * time.Second

// contextKey is an unexported type for context keys to avoid collisions.
type contextKey int

const principalKey contextKey = iota

// JWTAuthConfig configures the JWT authenticator.
type JWTAuthConfig struct {
	JWKSURL   string
	Audience  string
	Issuer    string
	ClockSkew time.Duration
	// AuthorizedClients, when non-empty, enforces that every token's client ID
	// (azp, client_id, or @clients subject suffix) is in this whitespace-separated
	// list. Applies to all token types — M2M and user alike.
	AuthorizedClients string
	// DisabledMockLocalPrincipal sets a static principal for local dev — empty in production.
	DisabledMockLocalPrincipal string
}

// JWTClaims extends standard JWT claims with LFX-specific fields.
type JWTClaims struct {
	jwt.RegisteredClaims
	Username        string `json:"https://sso.linuxfoundation.org/claims/username"`
	AuthorizedParty string `json:"azp"`
	ClientID        string `json:"client_id"`
	GrantType       string `json:"gty"`
	Email           string `json:"email"`
	EmailVerified   bool   `json:"email_verified"`
	Name            string `json:"name"`
	GivenName       string `json:"given_name"`
	FamilyName      string `json:"family_name"`
	Picture         string `json:"picture"`
}

const xUsernameHeader = "X-Username"
const xUserIDHeader = "X-User-ID"

// JWTAuthenticator validates JWTs using a JWKS endpoint.
type JWTAuthenticator struct {
	cfg               JWTAuthConfig
	keyfn             jwt.Keyfunc
	parser            *jwt.Parser
	logger            *slog.Logger
	authorizedClients map[string]struct{}
}

// NewJWTAuthenticator creates a JWTAuthenticator backed by the given JWKS URL.
// ctx controls the lifecycle of the background JWKS refresh goroutine.
// Set DisabledMockLocalPrincipal (without JWKSURL) to skip JWKS for local dev.
func NewJWTAuthenticator(ctx context.Context, cfg JWTAuthConfig, logger *slog.Logger) (*JWTAuthenticator, error) {
	if logger == nil {
		logger = slog.Default()
	}
	if cfg.DisabledMockLocalPrincipal != "" {
		if cfg.JWKSURL != "" {
			return nil, fmt.Errorf(
				"DISABLED_MOCK_LOCAL_PRINCIPAL and JWKS_URL are mutually exclusive: " +
					"remove DISABLED_MOCK_LOCAL_PRINCIPAL before deploying to an environment with a real JWKS endpoint",
			)
		}
		return &JWTAuthenticator{cfg: cfg, logger: logger, authorizedClients: buildClientSet(cfg.AuthorizedClients)}, nil
	}

	// Auth0 JWKS responses include x5t (SHA-1 thumbprint) fields that do not
	// round-trip through the jwkset validator cleanly, causing spurious
	// "X5T in marshal does not match X5T in marshalled" errors on every
	// refresh. ValidationSkipAll bypasses that structural check while still
	// enforcing cryptographic signature validation at token parse time.
	jwksProvider, err := keyfunc.NewDefaultOverrideCtx(ctx, []string{cfg.JWKSURL}, keyfunc.Override{
		ValidationSkipAll: true,
	})
	if err != nil {
		return nil, fmt.Errorf("fetch JWKS: %w", err)
	}
	parser := jwt.NewParser(
		jwt.WithAudience(cfg.Audience),
		jwt.WithIssuer(cfg.Issuer),
		jwt.WithLeeway(cfg.ClockSkew),
		jwt.WithExpirationRequired(),
		jwt.WithValidMethods([]string{"RS256", "RS384", "RS512", "ES256", "ES384", "ES512"}),
	)
	return &JWTAuthenticator{
		cfg:               cfg,
		keyfn:             jwksProvider.Keyfunc,
		parser:            parser,
		logger:            logger,
		authorizedClients: buildClientSet(cfg.AuthorizedClients),
	}, nil
}

// IsBypassActive reports whether JWT validation is bypassed (local dev only).
func (a *JWTAuthenticator) IsBypassActive() bool {
	return a.cfg.DisabledMockLocalPrincipal != ""
}

// Middleware returns an http.Handler middleware that validates the Bearer token
// and stores the Principal in the request context. Returns 401 on failure.
func (a *JWTAuthenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.cfg.DisabledMockLocalPrincipal != "" {
			ctx := ContextWithPrincipal(r.Context(), &models.Principal{
				UserID:        a.cfg.DisabledMockLocalPrincipal,
				Username:      a.cfg.DisabledMockLocalPrincipal,
				Email:         a.cfg.DisabledMockLocalPrincipal + "@local.dev",
				EmailVerified: true,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		token, err := a.extractAndValidate(r)
		if err != nil {
			a.logger.WarnContext(r.Context(), "auth: token validation failed", "error", err, "path", r.URL.Path)
			jsonError(w, http.StatusUnauthorized, "invalid or missing token")
			return
		}

		claims, ok := token.Claims.(*JWTClaims)
		if !ok {
			a.logger.WarnContext(r.Context(), "auth: unexpected claims type", "path", r.URL.Path)
			jsonError(w, http.StatusUnauthorized, "invalid token claims")
			return
		}

		if !a.isAuthorizedClient(claims) {
			a.logger.WarnContext(r.Context(), "auth: client not in authorized list", "path", r.URL.Path)
			jsonError(w, http.StatusUnauthorized, "invalid token claims")
			return
		}

		principalUserID := strings.TrimSpace(claims.Subject)
		principalUsername := strings.TrimSpace(claims.Username)
		if principalUsername == "" && a.isTrustedM2MForHeaderImpersonation(claims) {
			principalUsername = strings.TrimSpace(r.Header.Get(xUsernameHeader))
			if principalUsername != "" {
				// The acting user's Auth0 subject must accompany X-Username via the
				// companion X-User-ID header. Both are stripped by the ingress for
				// untrusted callers, so only this trusted M2M client can set them.
				principalUserID = strings.TrimSpace(r.Header.Get(xUserIDHeader))
			}
		}
		if principalUserID == "" {
			a.logger.WarnContext(r.Context(), "auth: empty subject in token", "path", r.URL.Path)
			jsonError(w, http.StatusUnauthorized, "invalid token claims")
			return
		}

		principal := &models.Principal{
			UserID:        principalUserID,
			Username:      principalUsername,
			Email:         claims.Email,
			EmailVerified: claims.EmailVerified,
			Name:          claims.Name,
			GivenName:     claims.GivenName,
			FamilyName:    claims.FamilyName,
			Picture:       claims.Picture,
		}
		next.ServeHTTP(w, r.WithContext(ContextWithPrincipal(r.Context(), principal)))
	})
}

// OptionalMiddleware is like Middleware but never rejects the request.
// If a valid token is present the Principal is stored in the context;
// if the token is absent or invalid the request continues with no principal.
func (a *JWTAuthenticator) OptionalMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.cfg.DisabledMockLocalPrincipal != "" {
			ctx := ContextWithPrincipal(r.Context(), &models.Principal{
				UserID:        a.cfg.DisabledMockLocalPrincipal,
				Username:      a.cfg.DisabledMockLocalPrincipal,
				Email:         a.cfg.DisabledMockLocalPrincipal + "@local.dev",
				EmailVerified: true,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		token, err := a.extractAndValidate(r)
		if err == nil {
			if claims, ok := token.Claims.(*JWTClaims); ok && claims.Subject != "" {
				principal := &models.Principal{
					UserID:        claims.Subject,
					Username:      claims.Username,
					Email:         claims.Email,
					EmailVerified: claims.EmailVerified,
					Name:          claims.Name,
					GivenName:     claims.GivenName,
					FamilyName:    claims.FamilyName,
					Picture:       claims.Picture,
				}
				r = r.WithContext(ContextWithPrincipal(r.Context(), principal))
			}
		}
		next.ServeHTTP(w, r)
	})
}

// isTrustedM2MForHeaderImpersonation reports whether the caller holds an M2M
// token from a client that has been explicitly listed in AuthorizedClients.
// When the list is empty the feature is disabled (false always returned), so
// an empty AUTHORIZED_CLIENTS disables X-Username impersonation entirely.
// User tokens (non-M2M) are never allowed to use the header regardless.
func (a *JWTAuthenticator) isTrustedM2MForHeaderImpersonation(claims *JWTClaims) bool {
	if len(a.authorizedClients) == 0 {
		return false // feature disabled
	}
	if !isM2MToken(claims) {
		return false // user tokens cannot use the header
	}
	actualClientID := strings.TrimSpace(getClientID(claims))
	_, ok := a.authorizedClients[actualClientID]
	return ok
}

// isAuthorizedClient reports whether the token's client ID is in the
// AuthorizedClients allowlist. When the list is empty all tokens are allowed.
// When non-empty, every token — M2M and user alike — must present a client ID
// that appears in the list. User tokens carry their SPA client ID in azp.
func (a *JWTAuthenticator) isAuthorizedClient(claims *JWTClaims) bool {
	if len(a.authorizedClients) == 0 {
		return true
	}
	actualClientID := strings.TrimSpace(getClientID(claims))
	if actualClientID == "" {
		return false
	}
	_, ok := a.authorizedClients[actualClientID]
	return ok
}

func isM2MToken(claims *JWTClaims) bool {
	if strings.TrimSpace(claims.GrantType) == "client_credentials" {
		return true
	}
	return strings.HasSuffix(strings.TrimSpace(claims.Subject), "@clients")
}

func getClientID(claims *JWTClaims) string {
	if azp := strings.TrimSpace(claims.AuthorizedParty); azp != "" {
		return azp
	}
	if clientID := strings.TrimSpace(claims.ClientID); clientID != "" {
		return clientID
	}
	// Only fall back to the @clients subject suffix for confirmed M2M tokens;
	// stripping this suffix from a user token subject would yield a nonsensical ID.
	if isM2MToken(claims) {
		sub := strings.TrimSpace(claims.Subject)
		return strings.TrimSuffix(sub, "@clients")
	}
	return ""
}

func (a *JWTAuthenticator) extractAndValidate(r *http.Request) (*jwt.Token, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, errors.New("missing Authorization header")
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return nil, errors.New("invalid Authorization header format")
	}
	raw := parts[1]

	claims := &JWTClaims{}
	token, err := a.parser.ParseWithClaims(raw, claims, a.keyfn)
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}
	return token, nil
}

// ContextWithPrincipal stores the principal in the context.
func ContextWithPrincipal(ctx context.Context, p *models.Principal) context.Context {
	return context.WithValue(ctx, principalKey, p)
}

// PrincipalFromContext retrieves the principal from the context.
// Returns nil if no principal is present.
func PrincipalFromContext(ctx context.Context) *models.Principal {
	p, _ := ctx.Value(principalKey).(*models.Principal)
	return p
}

// buildClientSet parses a whitespace-separated list of client IDs into a set
// for O(1) lookup. An empty string returns an empty (not nil) map.
func buildClientSet(s string) map[string]struct{} {
	ids := strings.Fields(s)
	set := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		set[id] = struct{}{}
	}
	return set
}

// jsonError writes a JSON {"error":"..."} response with the given status,
// matching the error shape used by all other API handlers.
func jsonError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(struct {
		Error string `json:"error"`
	}{Error: msg})
}
