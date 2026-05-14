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
	"strings"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

const (
	DefaultAudience  = "lfx-v2-initiatives-service"
	DefaultIssuer    = "heimdall"
	DefaultClockSkew = 5 * time.Second
)

// contextKey is an unexported type for context keys to avoid collisions.
type contextKey int

const principalKey contextKey = iota

// JWTAuthConfig configures the JWT authenticator.
type JWTAuthConfig struct {
	JWKSURL   string
	Audience  string
	Issuer    string
	ClockSkew time.Duration
	// DisabledMockLocalPrincipal sets a static principal for local dev — empty in production.
	DisabledMockLocalPrincipal string
}

// JWTClaims extends standard JWT claims with LFX-specific fields.
type JWTClaims struct {
	jwt.RegisteredClaims
	Email string `json:"email"`
	Name  string `json:"name"`
}

// JWTAuthenticator validates JWTs using a JWKS endpoint.
type JWTAuthenticator struct {
	cfg    JWTAuthConfig
	jwks   keyfunc.Keyfunc
	parser *jwt.Parser
}

// NewJWTAuthenticator creates and returns a JWTAuthenticator backed by the JWKS URL.
// When DisabledMockLocalPrincipal is set the JWKS fetch is skipped entirely —
// this allows local development without a real Auth0 domain.
//
// It is a configuration error to set both DisabledMockLocalPrincipal and JWKSURL
// at the same time; this prevents the bypass from silently winning in a deployment
// that also has a real JWKS URL configured.
func NewJWTAuthenticator(cfg JWTAuthConfig) (*JWTAuthenticator, error) {
	// Local dev bypass: skip remote JWKS fetch entirely.
	if cfg.DisabledMockLocalPrincipal != "" {
		// Reject ambiguous config: bypass + real JWKS URL set together is almost
		// certainly a misconfiguration (e.g. a staging deployment with a leftover
		// dev env var). Fail fast instead of silently bypassing auth.
		if cfg.JWKSURL != "" {
			return nil, fmt.Errorf(
				"DISABLED_MOCK_LOCAL_PRINCIPAL and JWKS_URL are mutually exclusive: " +
					"remove DISABLED_MOCK_LOCAL_PRINCIPAL before deploying to an environment with a real JWKS endpoint",
			)
		}
		return &JWTAuthenticator{cfg: cfg}, nil
	}

	jwks, err := keyfunc.NewDefaultCtx(context.Background(), []string{cfg.JWKSURL})
	if err != nil {
		return nil, fmt.Errorf("fetch JWKS: %w", err)
	}
	parser := jwt.NewParser(
		jwt.WithAudience(cfg.Audience),
		jwt.WithIssuer(cfg.Issuer),
		jwt.WithLeeway(cfg.ClockSkew),
		jwt.WithExpirationRequired(),
	)
	return &JWTAuthenticator{cfg: cfg, jwks: jwks, parser: parser}, nil
}

// Close releases JWKS resources.
func (a *JWTAuthenticator) Close() {
	// keyfunc v3 manages its own goroutines; no explicit close required.
}

// IsBypassActive reports whether the JWT bypass mode is enabled.
// Callers should log a prominent warning at startup when this returns true.
func (a *JWTAuthenticator) IsBypassActive() bool {
	return a.cfg.DisabledMockLocalPrincipal != ""
}

// Middleware returns an http.Handler middleware that validates the Bearer token
// and stores the Principal in the request context. Returns 401 on failure.
func (a *JWTAuthenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Local dev bypass — never set DisabledMockLocalPrincipal in production.
		if a.cfg.DisabledMockLocalPrincipal != "" {
			ctx := ContextWithPrincipal(r.Context(), &models.Principal{
				UserID: a.cfg.DisabledMockLocalPrincipal,
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

		principal := &models.Principal{
			UserID: claims.Subject,
			Email:  claims.Email,
			Name:   claims.Name,
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
	token, err := a.parser.ParseWithClaims(raw, claims, a.jwks.Keyfunc)
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

// jsonError writes a JSON {"error":"..."} response with the given status,
// matching the error shape used by all other API handlers.
func jsonError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(struct {
		Error string `json:"error"`
	}{Error: msg})
}
