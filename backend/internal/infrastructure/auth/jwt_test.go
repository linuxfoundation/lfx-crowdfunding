// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
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

func TestMiddleware_AcceptsRS256(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	parser := jwt.NewParser(
		jwt.WithAudience(testAudience),
		jwt.WithIssuer(testIssuer),
		jwt.WithExpirationRequired(),
		jwt.WithValidMethods([]string{"RS256", "RS384", "RS512", "ES256", "ES384", "ES512"}),
	)
	a := &JWTAuthenticator{
		cfg:    defaultCfg(),
		keyfn:  func(_ *jwt.Token) (any, error) { return &key.PublicKey, nil },
		parser: parser,
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodRS256, &JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "auth0|rs256user",
			Issuer:    testIssuer,
			Audience:  jwt.ClaimStrings{testAudience},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
		Username:      "rs256user",
		Email:         "rs256@example.com",
		EmailVerified: true,
	}).SignedString(key)
	if err != nil {
		t.Fatalf("sign RS256 token: %v", err)
	}

	var got *models.Principal
	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = PrincipalFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, makeRequest(tok))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for RS256 token, got %d", w.Code)
	}
	if got == nil {
		t.Fatal("principal not set in context")
	}
	if got.UserID != "auth0|rs256user" {
		t.Errorf("UserID = %q, want %q", got.UserID, "auth0|rs256user")
	}
	if got.Email != "rs256@example.com" {
		t.Errorf("Email = %q, want %q", got.Email, "rs256@example.com")
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

// ── middleware: enriched claims propagation ───────────────────────────────────

func TestMiddleware_EnrichedClaimsPropagated(t *testing.T) {
	a := newTestAuthenticator(defaultCfg())

	tok := sign(&JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "auth0|elim",
			Issuer:    testIssuer,
			Audience:  jwt.ClaimStrings{testAudience},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
		Username:      "elim",
		Email:         "elim@ds9.ufp",
		EmailVerified: true,
		GivenName:     "Elim",
		FamilyName:    "Garak",
		Picture:       "https://cdn.example.com/garak.png",
	})

	var got *models.Principal
	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = PrincipalFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, makeRequest(tok))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if got == nil {
		t.Fatal("principal not set in context")
	}
	if got.UserID != "auth0|elim" {
		t.Errorf("UserID = %q, want %q", got.UserID, "auth0|elim")
	}
	if got.Username != "elim" {
		t.Errorf("Username = %q, want %q", got.Username, "elim")
	}
	if got.Email != "elim@ds9.ufp" {
		t.Errorf("Email = %q, want %q", got.Email, "elim@ds9.ufp")
	}
	if !got.EmailVerified {
		t.Error("EmailVerified should be true")
	}
	if got.GivenName != "Elim" {
		t.Errorf("GivenName = %q, want %q", got.GivenName, "Elim")
	}
	if got.FamilyName != "Garak" {
		t.Errorf("FamilyName = %q, want %q", got.FamilyName, "Garak")
	}
	if got.Picture != "https://cdn.example.com/garak.png" {
		t.Errorf("Picture = %q, want %q", got.Picture, "https://cdn.example.com/garak.png")
	}
}

// ── middleware: empty subject rejected ───────────────────────────────────────

func TestMiddleware_EmptySubjectRejected(t *testing.T) {
	a := newTestAuthenticator(defaultCfg())

	// Token is valid but has no sub claim.
	tok := sign(&JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    testIssuer,
			Audience:  jwt.ClaimStrings{testAudience},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})

	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, makeRequest(tok))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for empty subject, got %d", w.Code)
	}
}

// ── middleware: malformed Authorization header ────────────────────────────────

func TestMiddleware_MalformedAuthHeader(t *testing.T) {
	a := newTestAuthenticator(defaultCfg())

	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	cases := []struct {
		name   string
		header string
	}{
		{"not Bearer scheme", "Token " + userToken()},
		{"single word only", "onlyone"},
		{"Basic auth", "Basic dXNlcjpwYXNz"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("Authorization", c.header)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != http.StatusUnauthorized {
				t.Errorf("expected 401, got %d", w.Code)
			}
		})
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

// TestNewJWTAuthenticator_EnforcesValidMethods exercises NewJWTAuthenticator
// end-to-end with a real in-process JWKS server, verifying that the constructor
// wires WithValidMethods correctly: HS256 must be rejected and RS256 accepted.
func TestNewJWTAuthenticator_EnforcesValidMethods(t *testing.T) {
	const kid = "test-key"

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}

	// Minimal JWKS endpoint backed by the generated public key.
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := base64.RawURLEncoding.EncodeToString(key.PublicKey.N.Bytes())
		e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.PublicKey.E)).Bytes())
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]any{{
				"kty": "RSA", "use": "sig", "kid": kid, "alg": "RS256",
				"n": n, "e": e,
			}},
		})
	}))
	defer jwksServer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	a, err := NewJWTAuthenticator(ctx, JWTAuthConfig{
		JWKSURL:   jwksServer.URL,
		Audience:  testAudience,
		Issuer:    testIssuer,
		ClockSkew: 0,
	})
	if err != nil {
		t.Fatalf("NewJWTAuthenticator: %v", err)
	}

	handler := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// HS256 token must be rejected by the asymmetric-only WithValidMethods list.
	t.Run("rejects HS256", func(t *testing.T) {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, makeRequest(userToken()))
		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 for HS256 token, got %d", w.Code)
		}
	})

	// RS256 token signed with the matching private key must be accepted.
	t.Run("accepts RS256", func(t *testing.T) {
		tok := jwt.NewWithClaims(jwt.SigningMethodRS256, &JWTClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   "auth0|e2euser",
				Issuer:    testIssuer,
				Audience:  jwt.ClaimStrings{testAudience},
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			},
			Email: "e2e@example.com",
		})
		tok.Header["kid"] = kid
		signed, err := tok.SignedString(key)
		if err != nil {
			t.Fatalf("sign RS256 token: %v", err)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, makeRequest(signed))
		if w.Code != http.StatusOK {
			t.Errorf("expected 200 for RS256 token via constructor, got %d", w.Code)
		}
	})
}
