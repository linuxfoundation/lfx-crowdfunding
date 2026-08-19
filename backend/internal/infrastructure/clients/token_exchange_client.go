// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package clients

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// TokenExchangeConfig holds settings for exchanging a Crowdfunding-audience user
// access token for a v2-platform-audience token via Auth0 Custom Token Exchange
// (RFC 8693), so v2 platform calls (e.g. query-service filter_grants=direct
// lookups) run as the calling user instead of a service identity.
type TokenExchangeConfig struct {
	// Auth0TokenURL is the Auth0 tenant's token endpoint, e.g.
	// https://linuxfoundation-dev.auth0.com/oauth/token
	Auth0TokenURL string
	// ClientID/ClientSecret authenticate the dedicated token-exchange M2M client
	// ("LFX Crowdfunding Token Exchange" in auth0-terraform).
	ClientID     string
	ClientSecret string
	// SubjectTokenType is the lfx_crowdfunding_api resource server identifier —
	// the audience of the incoming user token being exchanged.
	SubjectTokenType string
	// Audience is the lfx_v2_api resource server identifier — the audience of
	// the token minted by the exchange.
	Audience string
	// Timeout caps a single token-exchange HTTP call.
	Timeout time.Duration
}

// TokenExchangeClient exchanges a CF user's own access token for a v2-audience
// token for that same user.
type TokenExchangeClient interface {
	// Exchange returns a v2-audience access token for the user identified by
	// subjectToken. Callers must not log the returned token.
	Exchange(ctx context.Context, subjectToken string) (string, error)
}

type tokenExchangeHTTPClient struct {
	cfg        TokenExchangeConfig
	httpClient *http.Client

	mu    sync.Mutex
	cache map[string]cachedToken
}

type cachedToken struct {
	token  string
	expiry time.Time
}

// NewTokenExchangeClient returns a TokenExchangeClient, or nil when
// cfg.Auth0TokenURL is empty — callers treat a nil client as "token exchange
// disabled" (the v2-audience org lookup is skipped).
func NewTokenExchangeClient(cfg TokenExchangeConfig) TokenExchangeClient {
	if cfg.Auth0TokenURL == "" {
		return nil
	}
	return &tokenExchangeHTTPClient{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: cfg.Timeout},
		cache:      make(map[string]cachedToken),
	}
}

// auth0ExchangeResponse is the JSON body returned by Auth0's /oauth/token
// endpoint for a token-exchange grant.
type auth0ExchangeResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

func (c *tokenExchangeHTTPClient) Exchange(ctx context.Context, subjectToken string) (string, error) {
	key := cacheKey(subjectToken)

	c.mu.Lock()
	// ponytail: lazy sweep of expired entries bounds cache growth across the
	// process lifetime without a background goroutine; if the caller population
	// grows large enough for this scan to matter, swap for an LRU.
	now := time.Now()
	for k, v := range c.cache {
		if now.After(v.expiry) {
			delete(c.cache, k)
		}
	}
	if cached, ok := c.cache[key]; ok && now.Before(cached.expiry) {
		c.mu.Unlock()
		return cached.token, nil
	}
	c.mu.Unlock()

	form := url.Values{
		"grant_type":         {"urn:ietf:params:oauth:grant-type:token-exchange"},
		"subject_token":      {subjectToken},
		"subject_token_type": {c.cfg.SubjectTokenType},
		"audience":           {c.cfg.Audience},
		"client_id":          {c.cfg.ClientID},
		"client_secret":      {c.cfg.ClientSecret},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.Auth0TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("token exchange: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("token exchange: request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body) // drain so the connection can be reused
		return "", fmt.Errorf("token exchange: auth0 returned %d", resp.StatusCode)
	}
	var tr auth0ExchangeResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return "", fmt.Errorf("token exchange: decode response: %w", err)
	}
	if tr.AccessToken == "" {
		return "", fmt.Errorf("token exchange: auth0 returned empty access_token")
	}

	// Cache with a safety buffer so a near-expired token is never handed to the
	// downstream call. Clamp to half the TTL so short-lived tokens (≤120s)
	// don't produce a negative or zero duration.
	const bufferSec = 60
	buffer := time.Duration(bufferSec) * time.Second
	ttl := time.Duration(tr.ExpiresIn) * time.Second
	if ttl <= 2*buffer {
		buffer = ttl / 2
	}

	c.mu.Lock()
	c.cache[key] = cachedToken{token: tr.AccessToken, expiry: time.Now().Add(ttl - buffer)}
	c.mu.Unlock()

	return tr.AccessToken, nil
}

// cacheKey derives a cache key from a subject token without retaining the
// token itself in memory as a map key.
func cacheKey(subjectToken string) string {
	sum := sha256.Sum256([]byte(subjectToken))
	return hex.EncodeToString(sum[:])
}
