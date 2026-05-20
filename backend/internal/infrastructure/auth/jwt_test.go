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
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

// ── test constants ────────────────────────────────────────────────────────────

const (
	testSecret   = "test-secret-key-for-unit-tests-only"
	testAudience = "test-audience"
	testIssuer   = "test-issuer"
	testClientID = "abc123clientid"
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

func m2mToken(clientID, scope string) string {
	return sign(&JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   clientID + "@clients",
			Issuer:    testIssuer,
			Audience:  jwt.ClaimStrings{testAudience},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
		GrantType:       "client-credentials",
		AuthorizedParty: clientID,
		Scope:           scope,
	})
}

func makeRequest(token, username string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	if username != "" {
		r.Header.Set("X-Username", username)
	}
	return r
}

// ── middleware: user token path ───────────────────────────────────────────────

func TestMiddleware_UserToken(t *testing.T) {
	a := newTestAuthenticator(defaultCfg())

	var gotUserID, gotUsername string
	var gotIsM2M bool
	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := PrincipalFromContext(r.Context())
		if p != nil {
			gotUserID = p.UserID
			gotUsername = p.Username
			gotIsM2M = p.IsM2M
		}
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, makeRequest(userToken(), ""))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if gotUserID != "auth0|testuser" {
		t.Errorf("UserID = %q, want %q", gotUserID, "auth0|testuser")
	}
	if gotUsername != "testuser" {
		t.Errorf("Username = %q, want %q", gotUsername, "testuser")
	}
	if gotIsM2M {
		t.Error("IsM2M should be false for user token")
	}
}

// ── middleware: M2M path ──────────────────────────────────────────────────────

func TestMiddleware_M2M(t *testing.T) {
	m2mCfg := defaultCfg()
	m2mCfg.M2MScopeRequired = "access:api"
	m2mCfg.M2MAllowedClientIDs = []string{testClientID}

	tests := []struct {
		name       string
		token      string
		username   string
		wantStatus int
		wantM2M    bool
		wantUserID string
	}{
		{
			name:       "valid — scope and allowlist pass",
			token:      m2mToken(testClientID, "access:api"),
			username:   "elim",
			wantStatus: http.StatusOK,
			wantM2M:    true,
			wantUserID: "auth0|elim",
		},
		{
			name: "azp absent — client ID derived from sub",
			token: sign(&JWTClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject:   testClientID + "@clients",
					Issuer:    testIssuer,
					Audience:  jwt.ClaimStrings{testAudience},
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
				},
				GrantType: "client-credentials",
				// AuthorizedParty intentionally absent; sub-trim fallback should fire.
				Scope: "access:api",
			}),
			username:   "elim",
			wantStatus: http.StatusOK,
			wantM2M:    true,
			wantUserID: "auth0|elim",
		},
		{
			name:       "missing required scope",
			token:      m2mToken(testClientID, "other:scope"),
			username:   "elim",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "client ID not in allowlist",
			token:      m2mToken("unauthorized-client", "access:api"),
			username:   "elim",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "missing X-Username header",
			token:      m2mToken(testClientID, "access:api"),
			username:   "",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "X-Username contains pipe character",
			token:      m2mToken(testClientID, "access:api"),
			username:   "user|injected",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "X-Username contains space",
			token:      m2mToken(testClientID, "access:api"),
			username:   "bad user",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newTestAuthenticator(m2mCfg)

			var gotUserID string
			var gotIsM2M bool
			var gotClientID string
			h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				p := PrincipalFromContext(r.Context())
				if p != nil {
					gotUserID = p.UserID
					gotIsM2M = p.IsM2M
					gotClientID = p.M2MClientID
				}
				w.WriteHeader(http.StatusOK)
			}))

			w := httptest.NewRecorder()
			h.ServeHTTP(w, makeRequest(tt.token, tt.username))

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if tt.wantM2M {
				if !gotIsM2M {
					t.Error("IsM2M should be true")
				}
				if gotUserID != tt.wantUserID {
					t.Errorf("UserID = %q, want %q", gotUserID, tt.wantUserID)
				}
				if gotClientID != testClientID {
					t.Errorf("M2MClientID = %q, want %q", gotClientID, testClientID)
				}
			}
		})
	}
}

// ── middleware: token validation errors ───────────────────────────────────────

func TestMiddleware_NoToken(t *testing.T) {
	a := newTestAuthenticator(defaultCfg())
	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, makeRequest("", ""))
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
	h.ServeHTTP(w, makeRequest(expired, ""))
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
	h.ServeHTTP(w, makeRequest(tok, ""))
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
	h.ServeHTTP(w, makeRequest("", "")) // no token needed
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

// ── IsM2MPartiallyConfigured ──────────────────────────────────────────────────

func TestIsM2MPartiallyConfigured(t *testing.T) {
	tests := []struct {
		name  string
		scope string
		ids   []string
		want  bool
	}{
		// Both empty: not a partial config — M2M may simply not be in use.
		{"both empty", "", nil, false},
		// Exactly one set: inconsistent — warn.
		{"scope only", "access:api", nil, true},
		{"allowlist only", "", []string{"abc"}, true},
		// Both set: fully configured.
		{"both set", "access:api", []string{"abc"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &JWTAuthenticator{cfg: JWTAuthConfig{
				M2MScopeRequired:    tt.scope,
				M2MAllowedClientIDs: tt.ids,
			}}
			if got := a.IsM2MPartiallyConfigured(); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// ── hasScope ──────────────────────────────────────────────────────────────────

func TestHasScope(t *testing.T) {
	tests := []struct {
		scope  string
		target string
		want   bool
	}{
		{"access:api read:data", "access:api", true},
		{"access:api read:data", "write:data", false},
		{"", "access:api", false},
		{"access:api", "access:api", true},
		// Must be whole-word match, not substring.
		{"access:api-extended", "access:api", false},
	}
	for _, tt := range tests {
		if got := hasScope(tt.scope, tt.target); got != tt.want {
			t.Errorf("hasScope(%q, %q) = %v, want %v", tt.scope, tt.target, got, tt.want)
		}
	}
}

// ── RequireDirectAuth ─────────────────────────────────────────────────────────

func TestRequireDirectAuth(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("user token passes", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/", nil)
		r = r.WithContext(ContextWithPrincipal(r.Context(), &models.Principal{
			UserID: "auth0|elim",
			IsM2M:  false,
		}))
		w := httptest.NewRecorder()
		RequireDirectAuth(ok).ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	t.Run("M2M token rejected", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/", nil)
		r = r.WithContext(ContextWithPrincipal(r.Context(), &models.Principal{
			UserID:      "auth0|elim",
			IsM2M:       true,
			M2MClientID: testClientID,
		}))
		w := httptest.NewRecorder()
		RequireDirectAuth(ok).ServeHTTP(w, r)
		if w.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", w.Code)
		}
	})
}
