// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package auth

import (
	"context"
	"crypto"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

// ── test constants ────────────────────────────────────────────────────────────

const (
	testSecret   = "test-secret-key-for-unit-tests-only"
	testAudience = "test-audience"
	testIssuer   = "https://test-issuer.example"
)

var testRSAKey = mustGenerateRSAKey()

func mustGenerateRSAKey() *rsa.PrivateKey {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	return key
}

// ── test helpers ──────────────────────────────────────────────────────────────

// newTestAuthenticator builds an authenticator that validates RS256 tokens
// against an in-memory public key (no network JWKS fetch).
func newTestAuthenticator(cfg JWTAuthConfig) *JWTAuthenticator {
	return &JWTAuthenticator{
		cfg:               cfg,
		validator:         newJWTTestValidatorWithKey(cfg, &testRSAKey.PublicKey),
		logger:            slog.New(slog.NewTextHandler(io.Discard, nil)),
		authorizedClients: buildClientSet(cfg.AuthorizedClients),
	}
}

func newJWTTestValidatorWithKey(cfg JWTAuthConfig, publicKey *rsa.PublicKey) *validator.Validator {
	keyFunc := func(_ context.Context) (interface{}, error) {
		return publicKey, nil
	}

	v, err := validator.New(
		keyFunc,
		validator.RS256,
		cfg.Issuer,
		[]string{cfg.Audience},
		validator.WithCustomClaims(func() validator.CustomClaims { return &JWTClaims{} }),
		validator.WithAllowedClockSkew(cfg.ClockSkew),
	)
	if err != nil {
		panic(err)
	}

	return v
}

func defaultCfg() JWTAuthConfig {
	return JWTAuthConfig{
		Audience:  testAudience,
		Issuer:    testIssuer,
		ClockSkew: 5 * time.Second,
	}
}

// trustedM2MCfg returns a config with X-Username impersonation enabled for the
// "m2m-client" client (matching the subject used by m2mTokenWithoutUsername).
func trustedM2MCfg() JWTAuthConfig {
	cfg := defaultCfg()
	cfg.AuthorizedClients = "m2m-client"
	return cfg
}

func sign(claims map[string]any) string {
	signed, err := signRS256(claims, testRSAKey, "test-kid")
	if err != nil {
		panic(err)
	}
	return signed
}

func signHS256(claims map[string]any) string {
	signed, err := signHS256WithSecret(claims, []byte(testSecret))
	if err != nil {
		panic(err)
	}
	return signed
}

func signRS256(claims map[string]any, key *rsa.PrivateKey, kid string) (string, error) {
	header := map[string]any{"alg": "RS256", "typ": "JWT", "kid": kid}
	return signJWT(claims, header, func(signingInput string) ([]byte, error) {
		h := sha256.Sum256([]byte(signingInput))
		return rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, h[:])
	})
}

func signHS256WithSecret(claims map[string]any, secret []byte) (string, error) {
	header := map[string]any{"alg": "HS256", "typ": "JWT"}
	return signJWT(claims, header, func(signingInput string) ([]byte, error) {
		mac := hmac.New(sha256.New, secret)
		_, _ = mac.Write([]byte(signingInput))
		return mac.Sum(nil), nil
	})
}

func signJWT(claims map[string]any, header map[string]any, signer func(signingInput string) ([]byte, error)) (string, error) {
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	headerPart := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsPart := base64.RawURLEncoding.EncodeToString(claimsJSON)
	signingInput := headerPart + "." + claimsPart
	sig, err := signer(signingInput)
	if err != nil {
		return "", err
	}
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

func userTokenHS256() string {
	return signHS256(map[string]any{
		"sub": "auth0|testuser",
		"iss": testIssuer,
		"aud": testAudience,
		"exp": time.Now().Add(time.Hour).Unix(),
		"https://sso.linuxfoundation.org/claims/username": "testuser",
		"email":          "test@example.com",
		"email_verified": true,
		"given_name":     "Test",
		"family_name":    "User",
	})
}

func userToken() string {
	return sign(map[string]any{
		"sub": "auth0|testuser",
		"iss": testIssuer,
		"aud": testAudience,
		"exp": time.Now().Add(time.Hour).Unix(),
		"https://sso.linuxfoundation.org/claims/username": "testuser",
		"email":          "test@example.com",
		"email_verified": true,
		"given_name":     "Test",
		"family_name":    "User",
	})
}

func m2mTokenWithoutUsername() string {
	return sign(map[string]any{
		"sub":            "m2m-client@clients",
		"iss":            testIssuer,
		"aud":            testAudience,
		"exp":            time.Now().Add(time.Hour).Unix(),
		"azp":            "m2m-client",
		"gty":            "client_credentials",
		"email":          "",
		"email_verified": false,
	})
}

func m2mTokenForClient(clientID string) string {
	return sign(map[string]any{
		"sub": clientID + "@clients",
		"iss": testIssuer,
		"aud": testAudience,
		"exp": time.Now().Add(time.Hour).Unix(),
		"azp": clientID,
		"gty": "client_credentials",
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
	a := &JWTAuthenticator{
		cfg:               defaultCfg(),
		validator:         newJWTTestValidatorWithKey(defaultCfg(), &testRSAKey.PublicKey),
		logger:            slog.New(slog.NewTextHandler(io.Discard, nil)),
		authorizedClients: buildClientSet(""),
	}
	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// HS256 token must be rejected by the RS256-configured validator.
	w := httptest.NewRecorder()
	h.ServeHTTP(w, makeRequest(userTokenHS256()))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for HS256 token, got %d", w.Code)
	}
}

func TestMiddleware_AcceptsRS256(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	a := &JWTAuthenticator{
		cfg:               defaultCfg(),
		validator:         newJWTTestValidatorWithKey(defaultCfg(), &key.PublicKey),
		logger:            slog.New(slog.NewTextHandler(io.Discard, nil)),
		authorizedClients: buildClientSet(""),
	}
	claims := map[string]any{
		"sub": "auth0|rs256user",
		"iss": testIssuer,
		"aud": testAudience,
		"exp": time.Now().Add(time.Hour).Unix(),
		"https://sso.linuxfoundation.org/claims/username": "rs256user",
		"email":          "rs256@example.com",
		"email_verified": true,
	}
	signed, err := signRS256(claims, key, "test-kid")
	if err != nil {
		t.Fatalf("sign RS256 token: %v", err)
	}

	var got *models.Principal
	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = PrincipalFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, makeRequest(signed))
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
	expired := sign(map[string]any{
		"sub": "auth0|testuser",
		"iss": testIssuer,
		"aud": testAudience,
		"exp": time.Now().Add(-time.Hour).Unix(),
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
	tok := sign(map[string]any{
		"sub": "auth0|testuser",
		"iss": testIssuer,
		"aud": "wrong-audience",
		"exp": time.Now().Add(time.Hour).Unix(),
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

	tok := sign(map[string]any{
		"sub": "auth0|elim",
		"iss": testIssuer,
		"aud": testAudience,
		"exp": time.Now().Add(time.Hour).Unix(),
		"https://sso.linuxfoundation.org/claims/username": "elim",
		"email":          "elim@ds9.ufp",
		"email_verified": true,
		"given_name":     "Elim",
		"family_name":    "Garak",
		"picture":        "https://cdn.example.com/garak.png",
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

	// Token is valid but has no sub claim and no X-Username fallback.
	tok := sign(map[string]any{
		"iss": testIssuer,
		"aud": testAudience,
		"exp": time.Now().Add(time.Hour).Unix(),
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

// TestMiddleware_XUsernameIgnoredWhenFeatureDisabled verifies that an empty
// AUTHORIZED_CLIENTS (the default) disables X-Username impersonation
// entirely. A token with no sub and no M2M markers must be rejected even if
// the header is present.
func TestMiddleware_XUsernameIgnoredWhenFeatureDisabled(t *testing.T) {
	a := newTestAuthenticator(defaultCfg()) // AuthorizedClients is empty

	tok := sign(map[string]any{
		"iss": testIssuer,
		"aud": testAudience,
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := makeRequest(tok)
	r.Header.Set("X-Username", "acting-user")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 (feature disabled), got %d", w.Code)
	}
}

func TestMiddleware_UsesXUsernameHeaderWhenClaimMissing(t *testing.T) {
	a := newTestAuthenticator(trustedM2MCfg())

	var got *models.Principal
	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = PrincipalFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	r := makeRequest(m2mTokenWithoutUsername())
	r.Header.Set("X-Username", "acting-user")
	// X-User-ID header is no longer supported; UserID stays as the M2M token's subject.
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if got == nil {
		t.Fatal("principal not set in context")
	}
	if got.Username != "acting-user" {
		t.Errorf("Username = %q, want %q", got.Username, "acting-user")
	}
	// UserID is the M2M client's own subject (X-User-ID header is no longer read).
	if got.UserID != "m2m-client@clients" {
		t.Errorf("UserID = %q, want %q", got.UserID, "m2m-client@clients")
	}
}

func TestMiddleware_UsesCaseInsensitiveXUsernameHeaderWhenClaimMissing(t *testing.T) {
	a := newTestAuthenticator(trustedM2MCfg())

	cases := []struct {
		name       string
		headerName string
		value      string
	}{
		{name: "lowercase", headerName: "x-username", value: "lowercase-user"},
		{name: "uppercase", headerName: "X-USERNAME", value: "uppercase-user"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got *models.Principal
			h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got = PrincipalFromContext(r.Context())
				w.WriteHeader(http.StatusOK)
			}))

			r := makeRequest(m2mTokenWithoutUsername())
			r.Header.Set(tc.headerName, tc.value)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)

			if w.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d", w.Code)
			}
			if got == nil {
				t.Fatal("principal not set in context")
			}
			if got.Username != tc.value {
				t.Errorf("Username = %q, want %q", got.Username, tc.value)
			}
		})
	}
}

// Even when the feature is enabled, the JWT claim must win over the header.
// We use defaultCfg here (no AuthorizedClients) because userToken has no azp
// and the username-priority logic is independent of client gating.
func TestMiddleware_DoesNotOverrideClaimUsernameWithXUsernameHeader(t *testing.T) {
	a := newTestAuthenticator(defaultCfg())

	var gotUsername string
	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := PrincipalFromContext(r.Context())
		if p != nil {
			gotUsername = p.Username
		}
		w.WriteHeader(http.StatusOK)
	}))

	r := makeRequest(userToken())
	r.Header.Set("X-Username", "spoofed-user")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if gotUsername != "testuser" {
		t.Errorf("Username = %q, want %q", gotUsername, "testuser")
	}
}

// TestMiddleware_AcceptsImpersonationWithoutUserIDHeader verifies that
// supplying X-Username without X-User-ID is accepted. principalUserID
// stays as the M2M token's own claims.Subject in this case.
func TestMiddleware_AcceptsImpersonationWithoutUserIDHeader(t *testing.T) {
	a := newTestAuthenticator(trustedM2MCfg())

	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := makeRequest(m2mTokenWithoutUsername())
	r.Header.Set("X-Username", "acting-user")
	// X-User-ID intentionally absent — this is now allowed
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 when X-User-ID is absent, got %d", w.Code)
	}
}

func TestMiddleware_RejectsM2MTokenFromUnexpectedClient(t *testing.T) {
	cfg := defaultCfg()
	cfg.AuthorizedClients = "lfx-self-serve-client"
	a := newTestAuthenticator(cfg)

	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, makeRequest(m2mTokenForClient("some-other-client")))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestMiddleware_AcceptsM2MTokenFromExpectedClient(t *testing.T) {
	cfg := defaultCfg()
	cfg.AuthorizedClients = "lfx-self-serve-client"
	a := newTestAuthenticator(cfg)

	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, makeRequest(m2mTokenForClient("lfx-self-serve-client")))
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestMiddleware_AcceptsM2MTokenFromAnyExpectedClientInWhitespaceSeparatedList(t *testing.T) {
	cfg := defaultCfg()
	cfg.AuthorizedClients = "lfx-self-serve-client another-client\tthird-client\n"
	a := newTestAuthenticator(cfg)

	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	cases := []string{"lfx-self-serve-client", "another-client", "third-client"}
	for _, clientID := range cases {
		t.Run(clientID, func(t *testing.T) {
			w := httptest.NewRecorder()
			h.ServeHTTP(w, makeRequest(m2mTokenForClient(clientID)))
			if w.Code != http.StatusOK {
				t.Errorf("expected 200, got %d", w.Code)
			}
		})
	}
}

func TestMiddleware_RejectsM2MTokenWhenNotInExpectedClientList(t *testing.T) {
	cfg := defaultCfg()
	cfg.AuthorizedClients = "lfx-self-serve-client another-client"
	a := newTestAuthenticator(cfg)

	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, makeRequest(m2mTokenForClient("unlisted-client")))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

// TestMiddleware_AuthorizedClientsAppliesToUserTokens verifies that when
// AuthorizedClients is configured the client ID check applies to user tokens
// too, not only M2M tokens.
func TestMiddleware_AuthorizedClientsAppliesToUserTokens(t *testing.T) {
	cfg := defaultCfg()
	cfg.AuthorizedClients = "lfx-self-serve-client"
	a := newTestAuthenticator(cfg)

	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("user token with matching azp passes", func(t *testing.T) {
		tok := sign(map[string]any{
			"sub": "auth0|testuser",
			"iss": testIssuer,
			"aud": testAudience,
			"exp": time.Now().Add(time.Hour).Unix(),
			"https://sso.linuxfoundation.org/claims/username": "testuser",
			"azp": "lfx-self-serve-client",
		})
		w := httptest.NewRecorder()
		h.ServeHTTP(w, makeRequest(tok))
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	t.Run("user token without azp is rejected", func(t *testing.T) {
		// userToken() has no azp — no client ID can be extracted.
		w := httptest.NewRecorder()
		h.ServeHTTP(w, makeRequest(userToken()))
		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", w.Code)
		}
	})
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
		AllowMockPrincipalBypass:   true,
		DisabledMockLocalPrincipal: "user",
		JWKSURL:                    "https://example.com/.well-known/jwks.json",
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err == nil {
		t.Error("expected error when both bypass and JWKS_URL are set")
	}
}

func TestNewJWTAuthenticator_BypassRequiresExplicitAllowFlag(t *testing.T) {
	_, err := NewJWTAuthenticator(context.Background(), JWTAuthConfig{
		DisabledMockLocalPrincipal: "local-dev-user",
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err == nil {
		t.Fatal("expected constructor error when bypass allow flag is false")
	}
	if !strings.Contains(err.Error(), "ALLOW_MOCK_LOCAL_PRINCIPAL_BYPASS=true") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewJWTAuthenticator_WhitespaceBypassPrincipalDoesNotEnableBypass(t *testing.T) {
	_, err := NewJWTAuthenticator(context.Background(), JWTAuthConfig{
		DisabledMockLocalPrincipal: "   ",
		Audience:                   testAudience,
		Issuer:                     testIssuer,
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err == nil {
		t.Fatal("expected constructor error when bypass principal is whitespace-only")
	}
	if !strings.Contains(err.Error(), "JWKS_URL is required") {
		t.Fatalf("expected JWKS_URL required error, got: %v", err)
	}
}

func TestNewJWTAuthenticator_WhitespaceBypassPrincipalDoesNotBypassWithJWTConfig(t *testing.T) {
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"keys":[]}`))
	}))
	defer jwksServer.Close()

	a, err := NewJWTAuthenticator(context.Background(), JWTAuthConfig{
		DisabledMockLocalPrincipal: "   ",
		JWKSURL:                    jwksServer.URL,
		Audience:                   testAudience,
		Issuer:                     testIssuer,
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("NewJWTAuthenticator: %v", err)
	}
	if a.IsBypassActive() {
		t.Fatal("expected bypass mode to be disabled for whitespace-only principal")
	}

	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token when bypass is disabled, got %d", w.Code)
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
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("NewJWTAuthenticator: %v", err)
	}

	handler := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// HS256 token must be rejected by the asymmetric-only WithValidMethods list.
	t.Run("rejects HS256", func(t *testing.T) {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, makeRequest(userTokenHS256()))
		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 for HS256 token, got %d", w.Code)
		}
	})

	// RS256 token signed with the matching private key must be accepted.
	t.Run("accepts RS256", func(t *testing.T) {
		claims := map[string]any{
			"sub":   "auth0|e2euser",
			"iss":   testIssuer,
			"aud":   testAudience,
			"exp":   time.Now().Add(time.Hour).Unix(),
			"email": "e2e@example.com",
		}
		signed, err := signRS256(claims, key, kid)
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

func TestAuthFailureCategory(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "nil", err: nil, want: "unknown"},
		{name: "missing auth header", err: errMissingAuthorizationHeader, want: "missing_authorization_header"},
		{name: "malformed auth header", err: errInvalidAuthorizationHeader, want: "malformed_authorization_header"},
		{name: "missing bearer", err: errMissingBearerToken, want: "missing_bearer_token"},
		{name: "missing subject", err: errMissingSubjectClaim, want: "missing_subject"},
		{name: "context closed wrapped", err: fmt.Errorf("%w: %v", errAuthenticatorContextClosed, context.Canceled), want: "authenticator_context_closed"},
		{name: "expired string", err: errors.New("token is expired"), want: "token_expired"},
		{name: "audience string", err: errors.New("invalid audience"), want: "invalid_audience"},
		{name: "issuer string", err: errors.New("invalid issuer"), want: "invalid_issuer"},
		{name: "invalid token format sentinel", err: fmt.Errorf("invalid token format: %w", validator.ErrExcessiveTokenDots), want: "invalid_token_format"},
		{name: "signature", err: errors.New("signature verification failed"), want: "invalid_signature"},
		{name: "validator", err: errValidatorNotConfigured, want: "validator_not_configured"},
		{name: "fallback", err: errors.New("some unknown failure"), want: "token_validation_failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := authFailureCategory(tt.err)
			if got != tt.want {
				t.Fatalf("authFailureCategory() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ── M2M impersonation: middleware → handler guard integration ────────────────

// TestM2MImpersonation_PassesHandlerUsernameGuard verifies the full chain:
// a valid M2M token + X-Username header produces a principal that passes the
// handler-level "principal != nil && principal.Username != """ guard that all
// write handlers apply before delegating to a service.
func TestM2MImpersonation_PassesHandlerUsernameGuard(t *testing.T) {
	a := newTestAuthenticator(trustedM2MCfg())

	var capturedUsername string
	handlerReached := false

	// Inline handler that replicates the guard every write handler applies.
	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := PrincipalFromContext(r.Context())
		if p == nil || p.Username == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		handlerReached = true
		capturedUsername = p.Username
		w.WriteHeader(http.StatusOK)
	}))

	r := makeRequest(m2mTokenWithoutUsername())
	r.Header.Set("X-Username", "acting-user")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("M2M impersonation should pass the handler guard, got %d", w.Code)
	}
	if !handlerReached {
		t.Fatal("handler body was not reached")
	}
	if capturedUsername != "acting-user" {
		t.Errorf("Username = %q, want %q", capturedUsername, "acting-user")
	}
}

// TestM2MImpersonation_MissingXUsernameFailsHandlerGuard verifies that an M2M
// token without an X-Username header produces an empty Username, which is
// correctly rejected by the handler-level guard.
func TestM2MImpersonation_MissingXUsernameFailsHandlerGuard(t *testing.T) {
	a := newTestAuthenticator(trustedM2MCfg())

	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := PrincipalFromContext(r.Context())
		if p == nil || p.Username == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	r := makeRequest(m2mTokenWithoutUsername())
	// X-Username header intentionally absent — handler must reject.
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 when X-Username absent, got %d", w.Code)
	}
}

// ── MultiAudienceMiddleware ───────────────────────────────────────────────────

const testM2MAudience = "test-m2m-audience"

// newTestM2MAuthenticator builds a second authenticator that accepts tokens
// with testM2MAudience, simulating a dedicated M2M audience validator.
func newTestM2MAuthenticator() *JWTAuthenticator {
	cfg := JWTAuthConfig{
		Audience:          testM2MAudience,
		Issuer:            testIssuer,
		ClockSkew:         5 * time.Second,
		AuthorizedClients: "m2m-client",
	}
	return &JWTAuthenticator{
		cfg:               cfg,
		validator:         newJWTTestValidatorWithKey(cfg, &testRSAKey.PublicKey),
		logger:            slog.New(slog.NewTextHandler(io.Discard, nil)),
		authorizedClients: buildClientSet(cfg.AuthorizedClients),
	}
}

func m2mTokenWithM2MAudience() string {
	return sign(map[string]any{
		"sub": "m2m-client@clients",
		"iss": testIssuer,
		"aud": testM2MAudience,
		"exp": time.Now().Add(time.Hour).Unix(),
		"azp": "m2m-client",
		"gty": "client_credentials",
	})
}

// TestMultiAudienceMiddleware_PrimaryWins verifies that a valid user token (main
// audience) is accepted by MultiAudienceMiddleware even when a fallback is configured.
func TestMultiAudienceMiddleware_PrimaryWins(t *testing.T) {
	primary := newTestAuthenticator(defaultCfg())
	fallback := newTestM2MAuthenticator()
	mw := MultiAudienceMiddleware(primary, fallback)

	var gotUserID string
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if p := PrincipalFromContext(r.Context()); p != nil {
			gotUserID = p.UserID
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
}

// TestMultiAudienceMiddleware_FallbackWins verifies that an M2M token minted with the
// dedicated M2M audience is accepted when the primary (user audience) rejects it.
func TestMultiAudienceMiddleware_FallbackWins(t *testing.T) {
	primary := newTestAuthenticator(defaultCfg())
	fallback := newTestM2MAuthenticator()
	mw := MultiAudienceMiddleware(primary, fallback)

	var gotUserID string
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if p := PrincipalFromContext(r.Context()); p != nil {
			gotUserID = p.UserID
		}
		w.WriteHeader(http.StatusOK)
	}))

	r := makeRequest(m2mTokenWithM2MAudience())
	r.Header.Set("X-Username", "acting-user")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if gotUserID != "m2m-client@clients" {
		t.Errorf("UserID = %q, want %q", gotUserID, "m2m-client@clients")
	}
}

// TestMultiAudienceMiddleware_BothFail verifies that MultiAudienceMiddleware returns 401
// when the token is invalid for both authenticators.
func TestMultiAudienceMiddleware_BothFail(t *testing.T) {
	primary := newTestAuthenticator(defaultCfg())
	fallback := newTestM2MAuthenticator()
	mw := MultiAudienceMiddleware(primary, fallback)

	h := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Token signed for a completely different audience — neither accepts it.
	badToken := sign(map[string]any{
		"sub": "auth0|user",
		"iss": testIssuer,
		"aud": "totally-wrong-audience",
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	w := httptest.NewRecorder()
	h.ServeHTTP(w, makeRequest(badToken))

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

// TestMultiAudienceMiddleware_BypassModeUsedWhenActive verifies that MultiAudienceMiddleware
// uses the primary bypass path (local dev only) when it is active.
func TestMultiAudienceMiddleware_BypassModeUsedWhenActive(t *testing.T) {
	primaryCfg := defaultCfg()
	primaryCfg.DisabledMockLocalPrincipal = "local-dev-user"
	primary := &JWTAuthenticator{cfg: primaryCfg, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	fallback := newTestM2MAuthenticator()
	mw := MultiAudienceMiddleware(primary, fallback)

	var gotUserID string
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if p := PrincipalFromContext(r.Context()); p != nil {
			gotUserID = p.UserID
		}
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, makeRequest("")) // no token needed in bypass mode

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if gotUserID != "local-dev-user" {
		t.Errorf("UserID = %q, want %q", gotUserID, "local-dev-user")
	}
}
