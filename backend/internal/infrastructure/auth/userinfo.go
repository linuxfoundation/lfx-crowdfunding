// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ErrUserInfoTokenRejected is returned when the UserInfo endpoint rejects the
// access token (HTTP 4xx). The token has likely expired or been revoked.
var ErrUserInfoTokenRejected = errors.New("userinfo: access token rejected")

// ErrUserInfoUnavailable is returned when the UserInfo endpoint is unreachable
// or returns a 5xx response. The caller should surface this as a 503.
var ErrUserInfoUnavailable = errors.New("userinfo: service unavailable")

// UserInfoFetcher retrieves the authenticated user's profile from the OIDC
// UserInfo endpoint using the raw access token.
type UserInfoFetcher interface {
	FetchUserInfo(ctx context.Context, accessToken string) (*UserInfo, error)
}

// UserInfo holds the user profile returned by the OIDC UserInfo endpoint.
// Field names match the standard OIDC claims and the LF SSO namespaced claims
// injected by the Auth0 Action for this tenant.
type UserInfo struct {
	Sub        string `json:"sub"`
	Username   string `json:"https://sso.linuxfoundation.org/claims/username"`
	SSOEmail   string `json:"https://sso.linuxfoundation.org/claims/email"`
	Email      string `json:"email"`
	Name       string `json:"name"`
	GivenName  string `json:"given_name"`
	FamilyName string `json:"family_name"`
	Picture    string `json:"picture"`
}

// EffectiveEmail returns the best-available email address for this user info.
// The LF SSO namespaced claim takes precedence over the standard "email" claim.
func (u *UserInfo) EffectiveEmail() string {
	if v := strings.TrimSpace(u.SSOEmail); v != "" {
		return v
	}
	return strings.TrimSpace(u.Email)
}

// UserInfoClient calls the Auth0 OIDC UserInfo endpoint with the raw access
// token to retrieve the full user profile. This avoids embedding large profile
// claims in every access token (REQ-P2).
type UserInfoClient struct {
	endpoint   string
	httpClient *http.Client
}

// NewUserInfoClient creates a UserInfoClient that resolves the UserInfo URL
// from the given issuer (e.g. "https://example.auth0.com/").
func NewUserInfoClient(issuerURL string, httpClient *http.Client) (*UserInfoClient, error) {
	issuerURL = strings.TrimSuffix(strings.TrimSpace(issuerURL), "/")
	if issuerURL == "" {
		return nil, errors.New("issuer URL is required for UserInfo client")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &UserInfoClient{
		endpoint:   issuerURL + "/userinfo",
		httpClient: httpClient,
	}, nil
}

// FetchUserInfo calls the UserInfo endpoint and returns the decoded profile.
// The caller is responsible for passing the raw access token (without "Bearer ").
func (c *UserInfoClient) FetchUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build userinfo request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUserInfoUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Only true auth failures map to ErrUserInfoTokenRejected (forces re-auth).
		// Rate limiting (429), not-found (404), and server errors are retryable
		// upstream issues — classify them as ErrUserInfoUnavailable (503).
		switch resp.StatusCode {
		case http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden:
			return nil, fmt.Errorf("%w: HTTP %d", ErrUserInfoTokenRejected, resp.StatusCode)
		default:
			return nil, fmt.Errorf("%w: HTTP %d", ErrUserInfoUnavailable, resp.StatusCode)
		}
	}

	const maxBodyBytes = 64 * 1024 // 64 KB — well above any real OIDC UserInfo response
	var info UserInfo
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxBodyBytes)).Decode(&info); err != nil {
		return nil, fmt.Errorf("decode userinfo response: %w", err)
	}
	return &info, nil
}

// MockUserInfoFetcher returns a static UserInfo for local development and tests.
// Must never be used in production.
type MockUserInfoFetcher struct {
	username string
}

// NewMockUserInfoFetcher creates a fetcher that returns minimal profile data
// derived from the given username. Intended for local dev bypass mode.
func NewMockUserInfoFetcher(username string) *MockUserInfoFetcher {
	return &MockUserInfoFetcher{username: username}
}

// FetchUserInfo returns a static profile for local development.
// The access token is intentionally ignored — this fetcher is only activated
// when AllowMockPrincipalBypass=true, which must never be set in production
// or shared environments. In production, UserInfoClient is used instead.
func (m *MockUserInfoFetcher) FetchUserInfo(_ context.Context, _ string) (*UserInfo, error) {
	return &UserInfo{
		Sub:      m.username,
		Username: m.username,
		Email:    m.username + "@local.dev",
		Name:     "Local Dev User",
	}, nil
}
