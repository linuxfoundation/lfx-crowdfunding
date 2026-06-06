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
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/auth0/go-jwt-middleware/v2/jwks"
	"github.com/auth0/go-jwt-middleware/v2/validator"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

// DefaultClockSkew is the default leeway applied when validating JWT expiry.
const DefaultClockSkew = 5 * time.Second

// ScopeMe is the OAuth2 scope required for /me-style (user-issued token) routes.
const ScopeMe = "access:me"

// ScopeManage is the OAuth2 scope reserved for privileged admin/M2M routes.
// Currently unused in routing — no RequireScope(ScopeManage) route group exists yet.
// Bypass mode grants this scope so local dev is not broken when it is wired up.
const ScopeManage = "access:manage"

// contextKey is an unexported type for context keys to avoid collisions.
type contextKey int

const principalKey contextKey = iota

// JWTAuthConfig configures the JWT authenticator.
type JWTAuthConfig struct {
	JWKSURL   string
	Audience  string
	Issuer    string
	ClockSkew time.Duration
	// AllowMockPrincipalBypass must be true to permit DisabledMockLocalPrincipal.
	// Keep false in all shared/non-local environments.
	AllowMockPrincipalBypass bool
	// DisabledMockLocalPrincipal sets a static principal for local dev — empty in production.
	DisabledMockLocalPrincipal string
}

// JWTClaims extends standard JWT claims with LFX-specific fields.
type JWTClaims struct {
	Subject  string `json:"sub"`
	Username string `json:"https://sso.linuxfoundation.org/claims/username"`
	SSOEmail string `json:"https://sso.linuxfoundation.org/claims/email"`
	// Scope is a space-separated list of OAuth2 scopes granted to this token.
	// Route-group middleware checks for access:me or access:manage.
	Scope         string `json:"scope"`
	Email         string `json:"email"` // standard claim; prefer SSOEmail when both present
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
}

// effectiveEmail returns the canonical email address for the token.
// The LF SSO namespaced claim takes precedence over the standard "email" claim,
// matching how Auth0 injects profile data into access tokens.
func (c *JWTClaims) effectiveEmail() string {
	if v := strings.TrimSpace(c.SSOEmail); v != "" {
		return v
	}
	return strings.TrimSpace(c.Email)
}

const (
	authCategoryUnknown                    = "unknown"
	authCategoryMissingAuthorizationHeader = "missing_authorization_header"
	authCategoryMalformedAuthorization     = "malformed_authorization_header"
	authCategoryMissingBearerToken         = "missing_bearer_token"
	authCategoryMissingSubject             = "missing_subject"
	authCategoryContextClosed              = "authenticator_context_closed"
	authCategoryValidatorNotConfigured     = "validator_not_configured"
	authCategoryInvalidTokenFormat         = "invalid_token_format"
	authCategoryTokenExpired               = "token_expired"
	authCategoryInvalidAudience            = "invalid_audience"
	authCategoryInvalidIssuer              = "invalid_issuer"
	authCategoryInvalidSignature           = "invalid_signature"
	authCategoryTokenValidationFailed      = "token_validation_failed"
)

var (
	errMissingAuthorizationHeader = errors.New("missing Authorization header")
	errInvalidAuthorizationHeader = errors.New("invalid Authorization header format")
	errMissingBearerToken         = errors.New("missing bearer token")
	errMissingSubjectClaim        = errors.New("missing subject claim")
	errAuthenticatorContextClosed = errors.New("JWT authenticator context closed")
	errValidatorNotConfigured     = errors.New("JWT validator is not set up")
)

// JWTAuthenticator validates JWTs using a JWKS endpoint.
type JWTAuthenticator struct {
	cfg       JWTAuthConfig
	baseCtx   context.Context
	validator *validator.Validator
	logger    *slog.Logger
}

// NewJWTAuthenticator creates a JWTAuthenticator backed by the given JWKS URL.
// ctx gates the authenticator lifecycle; once canceled, token validation fails fast.
// Set DisabledMockLocalPrincipal (without JWKSURL) to skip JWKS for local dev.
func NewJWTAuthenticator(ctx context.Context, cfg JWTAuthConfig, logger *slog.Logger) (*JWTAuthenticator, error) {
	if logger == nil {
		logger = slog.Default()
	}
	if ctx == nil {
		ctx = context.Background()
	}
	// Normalize constructor inputs before wiring the provider/parser.
	jwksURLStr := strings.TrimSpace(cfg.JWKSURL)
	audience := strings.TrimSpace(cfg.Audience)
	issuer := strings.TrimSpace(cfg.Issuer)
	mockPrincipal := strings.TrimSpace(cfg.DisabledMockLocalPrincipal)
	clockSkew := cfg.ClockSkew
	if clockSkew == 0 {
		clockSkew = DefaultClockSkew
	}
	cfg.JWKSURL = jwksURLStr
	cfg.Audience = audience
	cfg.Issuer = issuer
	cfg.DisabledMockLocalPrincipal = mockPrincipal
	cfg.ClockSkew = clockSkew

	if mockPrincipal != "" {
		if !cfg.AllowMockPrincipalBypass {
			return nil, errors.New("DISABLED_MOCK_LOCAL_PRINCIPAL requires ALLOW_MOCK_LOCAL_PRINCIPAL_BYPASS=true")
		}
		if jwksURLStr != "" {
			return nil, fmt.Errorf(
				"DISABLED_MOCK_LOCAL_PRINCIPAL and JWKS_URL are mutually exclusive: " +
					"remove DISABLED_MOCK_LOCAL_PRINCIPAL before deploying to an environment with a real JWKS endpoint",
			)
		}
		return &JWTAuthenticator{
			cfg:     cfg,
			baseCtx: ctx,
			logger:  logger,
		}, nil
	}

	if audience == "" {
		return nil, errors.New("JWT_AUDIENCE is required")
	}
	if issuer == "" {
		return nil, errors.New("JWT_ISSUER is required")
	}
	if jwksURLStr == "" {
		return nil, errors.New("JWKS_URL is required")
	}

	issuerURL, err := url.Parse(issuer)
	if err != nil {
		return nil, fmt.Errorf("parse issuer URL: %w", err)
	}
	if !issuerURL.IsAbs() || issuerURL.Host == "" {
		return nil, errors.New("JWT_ISSUER must be an absolute URL")
	}
	if err := validateSecureURL(issuerURL, "JWT_ISSUER"); err != nil {
		return nil, err
	}
	jwksURL, err := url.Parse(jwksURLStr)
	if err != nil {
		return nil, fmt.Errorf("parse JWKS URL: %w", err)
	}
	if !jwksURL.IsAbs() || jwksURL.Host == "" {
		return nil, errors.New("JWKS_URL must be an absolute URL")
	}
	if err := validateSecureURL(jwksURL, "JWKS_URL"); err != nil {
		return nil, err
	}
	jwksProvider := jwks.NewCachingProvider(issuerURL, 5*time.Minute, jwks.WithCustomJWKSURI(jwksURL))
	keyFunc := func(reqCtx context.Context) (interface{}, error) {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("%w: %s", errAuthenticatorContextClosed, err)
		}
		ctxForKey, cancel := withValidatorRequestContext(ctx, reqCtx, 15*time.Second)
		defer cancel()
		return jwksProvider.KeyFunc(ctxForKey)
	}
	jwtValidator, err := validator.New(
		keyFunc,
		validator.RS256,
		issuer,
		[]string{audience},
		validator.WithCustomClaims(func() validator.CustomClaims { return &JWTClaims{} }),
		validator.WithAllowedClockSkew(clockSkew),
	)
	if err != nil {
		return nil, fmt.Errorf("build JWT validator: %w", err)
	}

	return &JWTAuthenticator{
		cfg:       cfg,
		baseCtx:   ctx,
		validator: jwtValidator,
		logger:    logger,
	}, nil
}

// Validate satisfies validator.CustomClaims.
func (c *JWTClaims) Validate(_ context.Context) error {
	// Use effectiveEmail so that the SSO email claim satisfies email_verified,
	// matching the precedence logic in effectiveEmail().
	if c.EmailVerified && c.effectiveEmail() == "" {
		return errors.New("email_verified requires email")
	}
	return nil
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
				// Grant both scopes in bypass mode so all route groups work locally.
				Scope: ScopeMe + " " + ScopeManage,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		claims, err := a.extractAndValidate(r)
		if err != nil {
			a.logger.WarnContext(r.Context(), "auth: token validation failed", "category", authFailureCategory(err), "error", err, "path", r.URL.Path)
			jsonError(w, http.StatusUnauthorized, "invalid or missing token")
			return
		}

		principalUserID := strings.TrimSpace(claims.Subject)
		principalUsername := strings.TrimSpace(claims.Username)
		if principalUserID == "" {
			a.logger.WarnContext(r.Context(), "auth: empty subject in token", "category", authCategoryMissingSubject, "path", r.URL.Path)
			jsonError(w, http.StatusUnauthorized, "invalid token claims")
			return
		}

		principal := &models.Principal{
			UserID:        principalUserID,
			Username:      principalUsername,
			Scope:         claims.Scope,
			Email:         claims.effectiveEmail(),
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
				Scope:         ScopeMe + " " + ScopeManage,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		claims, err := a.extractAndValidate(r)
		if err == nil {
			if claims != nil && claims.Subject != "" {
				principal := &models.Principal{
					UserID:        claims.Subject,
					Username:      claims.Username,
					Scope:         claims.Scope,
					Email:         claims.effectiveEmail(),
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

// RequireScope returns a middleware that enforces the given OAuth2 scope is
// present in the authenticated principal. Must be used after Middleware.
// Returns 403 Forbidden when the scope is absent.
func (a *JWTAuthenticator) RequireScope(scope string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p := PrincipalFromContext(r.Context())
			if p == nil {
				// Defensive: Middleware always runs first in production route groups,
				// so this branch is only reachable in misconfigured test setups.
				jsonError(w, http.StatusUnauthorized, "authentication required")
				return
			}
			if !hasScope(p.Scope, scope) {
				a.logger.WarnContext(r.Context(), "auth: insufficient scope",
					"required", scope, "path", r.URL.Path)
				jsonError(w, http.StatusForbidden, "insufficient scope")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// hasScope reports whether the space-separated scope string contains required.
func hasScope(scopeStr, required string) bool {
	for _, s := range strings.Fields(scopeStr) {
		if s == required {
			return true
		}
	}
	return false
}

func (a *JWTAuthenticator) extractAndValidate(r *http.Request) (*JWTClaims, error) {
	if a.baseCtx != nil {
		if err := a.baseCtx.Err(); err != nil {
			return nil, fmt.Errorf("%w: %s", errAuthenticatorContextClosed, err)
		}
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, errMissingAuthorizationHeader
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return nil, errInvalidAuthorizationHeader
	}
	raw := parts[1]
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errMissingBearerToken
	}

	if a.validator != nil {
		validated, err := a.validator.ValidateToken(r.Context(), raw)
		if err != nil {
			return nil, fmt.Errorf("validate token: %w", err)
		}
		validatedClaims, ok := validated.(*validator.ValidatedClaims)
		if !ok {
			return nil, errors.New("unexpected validated claims type")
		}
		claims, ok := validatedClaims.CustomClaims.(*JWTClaims)
		if !ok {
			return nil, errors.New("unexpected custom claims type")
		}
		claims.Subject = validatedClaims.RegisteredClaims.Subject
		if strings.TrimSpace(claims.Subject) == "" {
			return nil, errMissingSubjectClaim
		}
		return claims, nil
	}

	return nil, errValidatorNotConfigured
}

func withValidatorRequestContext(baseCtx context.Context, requestCtx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if requestCtx == nil {
		requestCtx = context.Background()
	}

	ctx := requestCtx
	cancelBase := func() {}
	if baseCtx != nil {
		ctx, cancelBase = context.WithCancel(ctx)
		if err := baseCtx.Err(); err != nil {
			cancelBase()
		} else {
			stop := context.AfterFunc(baseCtx, cancelBase)
			wrappedCancelBase := cancelBase
			cancelBase = func() {
				stop()
				wrappedCancelBase()
			}
		}
	}

	ctx, cancelTimeout := context.WithTimeout(ctx, timeout)
	return ctx, func() {
		cancelBase()
		cancelTimeout()
	}
}

func authFailureCategory(err error) string {
	if err == nil {
		return authCategoryUnknown
	}
	if errors.Is(err, errMissingAuthorizationHeader) {
		return authCategoryMissingAuthorizationHeader
	}
	if errors.Is(err, errInvalidAuthorizationHeader) {
		return authCategoryMalformedAuthorization
	}
	if errors.Is(err, errMissingBearerToken) {
		return authCategoryMissingBearerToken
	}
	if errors.Is(err, errMissingSubjectClaim) {
		return authCategoryMissingSubject
	}
	if errors.Is(err, errAuthenticatorContextClosed) {
		return authCategoryContextClosed
	}
	if errors.Is(err, errValidatorNotConfigured) {
		return authCategoryValidatorNotConfigured
	}
	if errors.Is(err, validator.ErrExcessiveTokenDots) {
		return authCategoryInvalidTokenFormat
	}

	errStr := strings.ToLower(err.Error())
	switch {
	case strings.Contains(errStr, "token is expired"):
		return authCategoryTokenExpired
	case strings.Contains(errStr, "invalid audience"):
		return authCategoryInvalidAudience
	case strings.Contains(errStr, "invalid issuer"):
		return authCategoryInvalidIssuer
	case strings.Contains(errStr, "invalid token format") || strings.Contains(errStr, "excessive dots"):
		return authCategoryInvalidTokenFormat
	case strings.Contains(errStr, "signature") || strings.Contains(errStr, "verification"):
		return authCategoryInvalidSignature
	default:
		return authCategoryTokenValidationFailed
	}
}

func validateSecureURL(u *url.URL, envName string) error {
	if strings.EqualFold(u.Scheme, "https") {
		return nil
	}
	if strings.EqualFold(u.Scheme, "http") && isLoopbackHost(u.Hostname()) {
		return nil
	}
	return fmt.Errorf("%s must use https (http is only allowed for loopback hosts)", envName)
}

func isLoopbackHost(host string) bool {
	h := strings.TrimSpace(strings.ToLower(host))
	if h == "localhost" {
		return true
	}
	ip := net.ParseIP(h)
	return ip != nil && ip.IsLoopback()
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

// jsonError writes a JSON {"error":"..."} response with the given status,
// matching the error shape used by all other API handlers.
func jsonError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(struct {
		Error string `json:"error"`
	}{Error: msg})
}
