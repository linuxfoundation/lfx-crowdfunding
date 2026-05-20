// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ── test constants ────────────────────────────────────────────────────────────

const (
	testSecret   = "test-secret-key-for-unit-tests-only"
	testAudience = "test-audience"
	testIssuer   = "test-issuer"
)

// ── test helpers ──────────────────────────────────────────────────────────────

// newTestAuthenticator builds an authenticator that validates HS256 tokens
// signed with testSecret, bypassing the JWKS fetch.
func newTestAuthenticator(cfg JWTAuthConfig) *JWTAuthenticator {
	secret := []byte(testSecret)
	parser := jwt.NewParser(
		jwt.WithAudience(cfg.Audience),
		jwt.WithIssuer(cfg.Issuer),
		jwt.WithLeeway(cfg.ClockSkew),
		jwt.WithExpirationRequired(),
	)
	return &JWTAuthenticator{
		cfg:    cfg,
		keyfn:  func(_ *jwt.Token) (any, error) { return secret, nil },
		parser: parser,
	}
}

func defaultCfg() JWTAuthConfig {
	return JWTAuthConfig{
		Audience:  testAudience,
		Issuer:    testIssuer,
		ClockSkew: 5 * time.Second,
	}
}

func sign(claims jwt.Claims) string {
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		panic(err)
	}
	return tok
}

func userToken() string {
	return sign(&JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "auth0|testuser",
			Issuer:    testIssuer,
			Audience:  jwt.ClaimStrings{testAudience},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
		Username:      "testuser",
		Email:         "test@example.com",
		EmailVerified: true,
		GivenName:     "Test",
		FamilyName:    "User",
	})
}

func makeRequest(token string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	return r
}

// ── middleware: user token path ───────────────────────────────────────────────

func TestMiddleware_UserToken(t *testing.T) {
	a := newTestAuthenticator(defaultCfg())

	var gotUserID, gotUsername string
	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := PrincipalFromContext(r.Context())
		if p != nil {
			gotUserID = p.UserID
			gotUsername = p.Username
		}
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, makeRequest(userToken()))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if gotUserID != "auth0|testuser" {
		t.Errorf("UserID = %q, want %q", gotUserID, "auth0|testuser")
	}
	if gotUsername != "testuser" {
		t.Errorf("Username = %q, want %q", gotUsername, "testuser")
	}
}

// ── middleware: algorithm restriction ───────────────────────────────────────

func TestMiddleware_RejectsHS256(t *testing.T) {
	// Build a parser restricted to asymmetric algorithms — matching production.
	// The keyfunc accepts any key so the only failure is the method check.
	parser := jwt.NewParser(
		jwt.WithAudience(testAudience),
		jwt.WithIssuer(testIssuer),
		jwt.WithExpirationRequired(),
		jwt.WithValidMethods([]string{"RS256", "RS384", "RS512", "ES256", "ES384", "ES512"}),
	)
	a := &JWTAuthenticator{
		cfg:    defaultCfg(),
		keyfn:  func(_ *jwt.Token) (any, error) { return []byte(testSecret), nil },
		parser: parser,
	}

	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// userToken() signs with HS256 — must be rejected by the asymmetric-only parser.
	w := httptest.NewRecorder()
	h.ServeHTTP(w, makeRequest(userToken()))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for HS256 token, got %d", w.Code)
	}
}

// ── middleware: token validation errors ────────────────────────────────────────

func TestMiddleware_NoToken(t *testing.T) {
	a := newTestAuthenticator(defaultCfg())
	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, makeRequest(""))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestMiddleware_ExpiredToken(t *testing.T) {
	a := newTestAuthenticator(defaultCfg())
	expired := sign(&JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "auth0|testuser",
			Issuer:    testIssuer,
			Audience:  jwt.ClaimStrings{testAudience},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		},
	})
	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, makeRequest(expired))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestMiddleware_WrongAudience(t *testing.T) {
	a := newTestAuthenticator(defaultCfg())
	tok := sign(&JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "auth0|testuser",
			Issuer:    testIssuer,
			Audience:  jwt.ClaimStrings{"wrong-audience"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})
	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, makeRequest(tok))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

// ── middleware: bypass mode ───────────────────────────────────────────────────

func TestMiddleware_BypassMode(t *testing.T) {
	cfg := defaultCfg()
	cfg.DisabledMockLocalPrincipal = "local-dev-user"
	a := &JWTAuthenticator{cfg: cfg}

	var gotUserID string
	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if p := PrincipalFromContext(r.Context()); p != nil {
			gotUserID = p.UserID
		}
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, makeRequest("")) // no token needed
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if gotUserID != "local-dev-user" {
		t.Errorf("UserID = %q, want %q", gotUserID, "local-dev-user")
	}
}

// ── constructor ───────────────────────────────────────────────────────────────

func TestNewJWTAuthenticator_MutualExclusion(t *testing.T) {
	_, err := NewJWTAuthenticator(context.Background(), JWTAuthConfig{
		DisabledMockLocalPrincipal: "user",
		JWKSURL:                    "https://example.com/.well-known/jwks.json",
	})
	if err == nil {
		t.Error("expected error when both bypass and JWKS_URL are set")
	}
}
