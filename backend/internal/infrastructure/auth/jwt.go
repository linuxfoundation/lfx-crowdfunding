// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package auth provides JWT validation for Auth0-issued tokens.
package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

// DefaultAudience, DefaultIssuer, and DefaultClockSkew define the default JWT validation settings.
const (
	DefaultAudience  = "lfx-v2-initiatives-service"
	DefaultIssuer    = "heimdall"
	DefaultClockSkew = 5 * time.Second
)

// userIDPrefix is prepended to the LF username to form the user_id stored in
// the crowdfunding database (e.g. "auth0|elim"). This matches the format used
// by direct user logins, so M2M-proxied and direct requests resolve the same row.
const userIDPrefix = "auth0|"

// headerXUsername is the request header sent by M2M callers to identify the
// target user when the JWT itself carries no user-specific claims.
const headerXUsername = "X-Username"

// usernameRe validates X-Username values; prevents pipe injection into the constructed UserID.
var usernameRe = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

// contextKey is an unexported type for context keys to avoid collisions.
type contextKey int

const principalKey contextKey = iota

// JWTAuthConfig configures the JWT authenticator.
type JWTAuthConfig struct {
	JWKSURL   string
	Audience  string
	Issuer    string
	ClockSkew time.Duration
	// M2MScopeRequired, when non-empty, requires M2M tokens to carry this scope
	// in the "scope" claim before the X-Username header is trusted. This limits
	// the impersonation surface to M2M clients that have been explicitly granted
	// the scope in Auth0 (e.g. "access:api").
	M2MScopeRequired string
	// M2MAllowedClientIDs, when non-empty, is an explicit allowlist of Auth0
	// client IDs permitted to use the M2M proxy path. The value is read from
	// the token's "azp" claim, with the "sub" claim used as a fallback if
	// "azp" is absent. When empty the allowlist check is skipped (not
	// recommended for production).
	M2MAllowedClientIDs []string
	// DisabledMockLocalPrincipal sets a static principal for local dev — empty in production.
	DisabledMockLocalPrincipal string
}

// JWTClaims extends standard JWT claims with LFX-specific fields.
type JWTClaims struct {
	jwt.RegisteredClaims
	Username      string `json:"https://sso.linuxfoundation.org/claims/username"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	// Scope is a space-separated list of scopes present in M2M client-credentials tokens.
	Scope string `json:"scope"`
	// GrantType is the Auth0 grant type used to obtain the token.
	// "client-credentials" identifies M2M tokens; absent or empty on ID tokens.
	GrantType string `json:"gty"`
	// AuthorizedParty is the Auth0 client_id of the application that requested
	// the token (OIDC "azp" claim). Present on M2M client-credentials tokens;
	// equals the client_id portion of the "sub" claim.
	AuthorizedParty string `json:"azp"`
}

// JWTAuthenticator validates JWTs using a JWKS endpoint.
type JWTAuthenticator struct {
	cfg    JWTAuthConfig
	keyfn  jwt.Keyfunc
	parser *jwt.Parser
}

// NewJWTAuthenticator creates a JWTAuthenticator backed by the given JWKS URL.
// ctx controls the lifecycle of the background JWKS refresh goroutine.
// Set DisabledMockLocalPrincipal (without JWKSURL) to skip JWKS for local dev.
func NewJWTAuthenticator(ctx context.Context, cfg JWTAuthConfig) (*JWTAuthenticator, error) {
	if cfg.DisabledMockLocalPrincipal != "" {
		if cfg.JWKSURL != "" {
			return nil, fmt.Errorf(
				"DISABLED_MOCK_LOCAL_PRINCIPAL and JWKS_URL are mutually exclusive: " +
					"remove DISABLED_MOCK_LOCAL_PRINCIPAL before deploying to an environment with a real JWKS endpoint",
			)
		}
		return &JWTAuthenticator{cfg: cfg}, nil
	}

	jwksProvider, err := keyfunc.NewDefaultCtx(ctx, []string{cfg.JWKSURL})
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
	return &JWTAuthenticator{cfg: cfg, keyfn: jwksProvider.Keyfunc, parser: parser}, nil
}

// Close is a no-op; keyfunc v3 goroutines are stopped via the context.
func (a *JWTAuthenticator) Close() {}

// IsBypassActive reports whether JWT validation is bypassed (local dev only).
func (a *JWTAuthenticator) IsBypassActive() bool {
	return a.cfg.DisabledMockLocalPrincipal != ""
}

// IsM2MPartiallyConfigured reports whether exactly one of M2MScopeRequired /
// M2MAllowedClientIDs is set. Both should be set together in production.
func (a *JWTAuthenticator) IsM2MPartiallyConfigured() bool {
	hasScope := a.cfg.M2MScopeRequired != ""
	hasAllowlist := len(a.cfg.M2MAllowedClientIDs) > 0
	return hasScope != hasAllowlist // XOR: exactly one is set
}

// Middleware returns an http.Handler middleware that validates the Bearer token
// and stores the Principal in the request context. Returns 401 on failure.
func (a *JWTAuthenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.cfg.DisabledMockLocalPrincipal != "" {
			ctx := ContextWithPrincipal(r.Context(), &models.Principal{
				UserID:        a.cfg.DisabledMockLocalPrincipal,
				Username:      a.cfg.DisabledMockLocalPrincipal,
				EmailVerified: true,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		token, err := a.extractAndValidate(r)
		if err != nil {
			jsonError(w, http.StatusUnauthorized, "invalid or missing token")
			return
		}

		claims, ok := token.Claims.(*JWTClaims)
		if !ok || claims.Subject == "" {
			jsonError(w, http.StatusUnauthorized, "invalid token claims")
			return
		}

		var principal *models.Principal
		if claims.GrantType != "client-credentials" {
			// User ID token: populate principal from JWT claims.
			principal = &models.Principal{
				UserID:        claims.Subject,
				Username:      claims.Username,
				Email:         claims.Email,
				EmailVerified: claims.EmailVerified,
				Name:          claims.Name,
				GivenName:     claims.GivenName,
				FamilyName:    claims.FamilyName,
				Picture:       claims.Picture,
			}
		} else {
			// M2M client-credentials token: trust X-Username only after scope,
			// allowlist, and format checks.
			if a.cfg.M2MScopeRequired != "" && !hasScope(claims.Scope, a.cfg.M2MScopeRequired) {
				jsonError(w, http.StatusForbidden, "M2M token missing required scope")
				return
			}
			// Prefer azp (client_id without suffix) over trimming sub.
			// When azp is absent, only accept a sub that carries the
			// expected "@clients" suffix and produces a non-empty ID;
			// anything else indicates a malformed or unexpected token.
			clientID := claims.AuthorizedParty
			if clientID == "" {
				if !strings.HasSuffix(claims.Subject, "@clients") {
					jsonError(w, http.StatusUnauthorized, "M2M token missing azp claim and sub does not end with @clients")
					return
				}
				clientID = strings.TrimSuffix(claims.Subject, "@clients")
				if clientID == "" {
					jsonError(w, http.StatusUnauthorized, "M2M token has empty client ID")
					return
				}
			}
			if len(a.cfg.M2MAllowedClientIDs) > 0 && !hasAllowedClientID(a.cfg.M2MAllowedClientIDs, clientID) {
				jsonError(w, http.StatusForbidden, "M2M client not permitted to proxy user requests")
				return
			}
			username := strings.TrimSpace(r.Header.Get(headerXUsername))
			if username == "" {
				jsonError(w, http.StatusUnauthorized, "M2M token requires X-Username header")
				return
			}
			if !usernameRe.MatchString(username) {
				jsonError(w, http.StatusBadRequest, "X-Username contains invalid characters")
				return
			}
			principal = &models.Principal{
				UserID:      userIDPrefix + username,
				Username:    username,
				IsM2M:       true,
				M2MClientID: clientID,
			}
		}
		next.ServeHTTP(w, r.WithContext(ContextWithPrincipal(r.Context(), principal)))
	})
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

// hasScope reports whether the space-separated scope string contains target.
func hasScope(scope, target string) bool {
	for _, s := range strings.Fields(scope) {
		if s == target {
			return true
		}
	}
	return false
}

// hasAllowedClientID reports whether clientID is in the allowlist.
func hasAllowedClientID(allowed []string, clientID string) bool {
	for _, id := range allowed {
		if id == clientID {
			return true
		}
	}
	return false
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

// RequireDirectAuth is middleware that rejects M2M-proxied requests.
// Apply it to routes where the caller must hold their own token (e.g. payment
// operations that create Stripe customers and require a real user email).
func RequireDirectAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := PrincipalFromContext(r.Context())
		if p != nil && p.IsM2M {
			jsonError(w, http.StatusForbidden, "payment operations require a user token, not an M2M token")
			return
		}
		next.ServeHTTP(w, r)
	})
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
