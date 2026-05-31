// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1" //nolint:gosec // SHA-1 is required by RFC 7517 for the x5t thumbprint
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
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
		cfg:               cfg,
		keyfn:             func(_ *jwt.Token) (any, error) { return secret, nil },
		parser:            parser,
		logger:            slog.New(slog.NewTextHandler(io.Discard, nil)),
		authorizedClients: buildClientSet(cfg.AuthorizedClients),
	}
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

func m2mTokenWithoutUsername() string {
	return sign(&JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "m2m-client@clients",
			Issuer:    testIssuer,
			Audience:  jwt.ClaimStrings{testAudience},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
		AuthorizedParty: "m2m-client",
		GrantType:       "client_credentials",
		Email:           "",
		EmailVerified:   false,
	})
}

func m2mTokenForClient(clientID string) string {
	return sign(&JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   clientID + "@clients",
			Issuer:    testIssuer,
			Audience:  jwt.ClaimStrings{testAudience},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
		AuthorizedParty: clientID,
		GrantType:       "client_credentials",
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
		cfg:               defaultCfg(),
		keyfn:             func(_ *jwt.Token) (any, error) { return []byte(testSecret), nil },
		parser:            parser,
		logger:            slog.New(slog.NewTextHandler(io.Discard, nil)),
		authorizedClients: buildClientSet(""),
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
		cfg:               defaultCfg(),
		keyfn:             func(_ *jwt.Token) (any, error) { return &key.PublicKey, nil },
		parser:            parser,
		logger:            slog.New(slog.NewTextHandler(io.Discard, nil)),
		authorizedClients: buildClientSet(""),
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

	// Token is valid but has no sub claim and no X-Username fallback.
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

// TestMiddleware_XUsernameIgnoredWhenFeatureDisabled verifies that an empty
// AUTHORIZED_CLIENTS (the default) disables X-Username impersonation
// entirely. A token with no sub and no M2M markers must be rejected even if
// the header is present.
func TestMiddleware_XUsernameIgnoredWhenFeatureDisabled(t *testing.T) {
	a := newTestAuthenticator(defaultCfg()) // AuthorizedClients is empty

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
	r.Header.Set("X-User-ID", "auth0|abc123")
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
	if got.UserID != "auth0|abc123" {
		t.Errorf("UserID = %q, want %q", got.UserID, "auth0|abc123")
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
			r.Header.Set("X-User-ID", "auth0|caseuser")
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
	r.Header.Set("X-User-ID", "auth0|spoofed")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if gotUsername != "testuser" {
		t.Errorf("Username = %q, want %q", gotUsername, "testuser")
	}
}

// TestMiddleware_RejectsImpersonationWithoutUserIDHeader verifies that
// supplying X-Username without the companion X-User-ID header is rejected.
// The acting user's real Auth0 subject must always accompany the username.
func TestMiddleware_RejectsImpersonationWithoutUserIDHeader(t *testing.T) {
	a := newTestAuthenticator(trustedM2MCfg())

	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := makeRequest(m2mTokenWithoutUsername())
	r.Header.Set("X-Username", "acting-user")
	// X-User-ID intentionally absent
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 when X-User-ID is absent, got %d", w.Code)
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
		tok := sign(&JWTClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   "auth0|testuser",
				Issuer:    testIssuer,
				Audience:  jwt.ClaimStrings{testAudience},
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			},
			Username:        "testuser",
			AuthorizedParty: "lfx-self-serve-client",
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
		DisabledMockLocalPrincipal: "user",
		JWKSURL:                    "https://example.com/.well-known/jwks.json",
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
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

// TestNewJWTAuthenticator_AcceptsJWKSWithX5T is a regression test for the Auth0
// JWKS x5t handling. Auth0's JWKS responses include x5c (certificate chain) and
// x5t (SHA-1 thumbprint) fields. The jwkset validator recomputes the thumbprint
// and rejects the set with "X5T in marshal does not match X5T in marshalled" on
// every refresh unless the constructor passes ValidationSkipAll. When that
// happens no signing keys load and every token is rejected as
// "invalid or missing token" (401). This test stands up an Auth0-style JWKS
// endpoint with x5c/x5t and asserts a token signed with that key is accepted —
// reproducing the failure a refactor previously introduced by dropping the
// override.
func TestNewJWTAuthenticator_AcceptsJWKSWithX5T(t *testing.T) {
	const kid = "test-key-x5t"

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}

	// Build a self-signed certificate so the JWKS can carry x5c/x5t, mirroring
	// the shape of a real Auth0 JWKS entry.
	certDER, err := x509.CreateCertificate(
		rand.Reader,
		&x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "test"}},
		&x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "test"}},
		&key.PublicKey,
		key,
	)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	thumbprint := sha1.Sum(certDER) //nolint:gosec // x5t is defined as the SHA-1 thumbprint

	// Auth0 encodes x5t as base64url(uppercase-hex(sha1(cert))), whereas the
	// jwkset library computes base64url(raw sha1 bytes). The two never match, so
	// jwkset rejects the key set unless ValidationSkipAll is set. Emit x5t exactly
	// the way Auth0 does so this test reproduces the real-world failure. (x5t is
	// defined as base64url in RFC 7517, so use RawURLEncoding for the outer step.)
	auth0X5T := base64.RawURLEncoding.EncodeToString(
		[]byte(strings.ToUpper(hex.EncodeToString(thumbprint[:]))),
	)

	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := base64.RawURLEncoding.EncodeToString(key.PublicKey.N.Bytes())
		e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.PublicKey.E)).Bytes())
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]any{{
				"kty": "RSA", "use": "sig", "kid": kid, "alg": "RS256",
				"n": n, "e": e,
				"x5c": []string{base64.StdEncoding.EncodeToString(certDER)},
				"x5t": auth0X5T,
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

	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, &JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "auth0|x5tuser",
			Issuer:    testIssuer,
			Audience:  jwt.ClaimStrings{testAudience},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
		Email: "x5t@example.com",
	})
	tok.Header["kid"] = kid
	signed, err := tok.SignedString(key)
	if err != nil {
		t.Fatalf("sign RS256 token: %v", err)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, makeRequest(signed))
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for token validated against JWKS with x5t/x5c, got %d", w.Code)
	}
}
