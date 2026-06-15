// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package clients provides outbound HTTP clients for external services.
package clients

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/core"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var reimbursementTracer = otel.Tracer("reimbursement-client")

// rsHTTPError is a typed error returned when the Reimbursement Service responds
// with an HTTP error status. Using a sentinel type instead of a formatted string
// lets callers reliably inspect the status code (e.g. to distinguish 404 from
// other errors) without fragile string-prefix matching.
type rsHTTPError struct{ code int }

func (e *rsHTTPError) Error() string { return fmt.Sprintf("reimbursement service returned %d", e.code) }

// reimbursementContact holds a single contact entry (owner or beneficiary).
// JSON keys must match the Reimbursement Service API contract exactly.
type reimbursementContact struct {
	EmailAddress string `json:"EmailAddress"`
	Name         string `json:"Name"`
}

// rsPolicyUpdate is the PATCH /reimbursement/{id} request body.
// Matches the legacy ReimbursementUpdate struct in the LFF reimbursementservice.
type rsPolicyUpdate struct {
	Beneficiaries []reimbursementContact `json:"Beneficiaries"` // no omitempty — empty array must be sent to clear beneficiaries
	Categories    []string               `json:"Categories"`
	Owner         reimbursementContact   `json:"Owner"`
	ProjectURL    string                 `json:"ProjectURL"`
	EntityType    string                 `json:"EntityType,omitempty"`
}

// rsPolicyCreate is the POST /reimbursement/{id} request body.
// Embeds rsPolicyUpdate and adds ProjectName.
// Matches the legacy ReimbursementCreate struct in the LFF reimbursementservice.
type rsPolicyCreate struct {
	rsPolicyUpdate
	ProjectName string `json:"ProjectName"`
}

// ReimbursementClient syncs initiative policies (owner, beneficiaries, categories)
// with the Expensify-backed Reimbursement Service whenever an initiative changes.
type ReimbursementClient interface {
	// SyncPolicy upserts the initiative's reimbursement policy in the RS.
	// It is a no-op when the initiative is not published.
	// Errors are non-fatal by convention — callers log at warn and continue.
	SyncPolicy(ctx context.Context, initiative *models.Initiative, ownerUser *models.User) error

	// ProcessExpenseAction submits an action (e.g. "approve", "reject") against
	// the given expense report in the Reimbursement Service.
	// Maps upstream 404 → domain.ErrExpenseReportNotFound so callers can
	// distinguish missing reports from other upstream errors.
	ProcessExpenseAction(ctx context.Context, action, reportID string) error
}

// ReimbursementConfig holds all connection settings for the Reimbursement Service.
type ReimbursementConfig struct {
	// APIURL is the base URL of the Reimbursement Service, e.g.
	// https://rs.example.com — a trailing slash is normalised at call time.
	APIURL string

	// APIKey is the static pre-shared secret sent in the X-API-KEY header.
	// Corresponds to the RS_API_KEY environment variable on the service side.
	APIKey string

	// FrontendBase is the public base URL of the frontend
	// (e.g. https://crowdfunding.lfx.linuxfoundation.org).
	// Used to construct the initiative's public URL in the policy payload.
	FrontendBase string

	// Timeout caps individual outbound HTTP calls, including Auth0 token
	// fetches and Reimbursement Service API requests.
	Timeout time.Duration

	// --- Optional Auth0 private-key JWT (M2M) config ----------------------
	// The API gateway in front of the RS requires a Bearer token on all routes.
	// When all four fields below are set the client signs a JWT assertion with
	// the RSA private key, exchanges it for an access token via the Auth0
	// client_credentials grant, caches it, and attaches it as
	// Authorization: Bearer on every RS call alongside X-API-KEY.
	// Leave empty only if the gateway is configured to accept X-API-KEY alone.

	// Auth0TokenURL is the token endpoint, e.g.
	// https://linuxfoundation-dev.auth0.com/oauth/token
	Auth0TokenURL string
	// Auth0ClientID is the M2M application client ID.
	Auth0ClientID string
	// Auth0ClientPrivateKey is the PEM-encoded RSA private key used to sign
	// the JWT assertion. Must be PKCS8 or PKCS1 PEM.
	Auth0ClientPrivateKey string
	// Auth0Audience is the resource server audience the gateway validates against,
	// e.g. https://api-gw.dev.platform.linuxfoundation.org/
	Auth0Audience string
}

type reimbursementHTTPClient struct {
	cfg         ReimbursementConfig
	httpClient  *core.HTTPClient
	tokenClient *http.Client // used only for Auth0 token fetches
	tokenMu     sync.Mutex
	tokenVal    string
	tokenExpiry time.Time
	// private key is parsed once from Auth0ClientPrivateKey PEM on first use.
	keyOnce    sync.Once
	privateKey *rsa.PrivateKey
	keyErr     error
}

// NewReimbursementClient creates a ReimbursementClient from the given config.
// Returns nil when cfg.APIURL is empty — the service layer treats a nil client
// as a disabled RS integration (no sync, no startup error).
func NewReimbursementClient(cfg ReimbursementConfig) ReimbursementClient {
	if cfg.APIURL == "" {
		return nil
	}
	return &reimbursementHTTPClient{
		cfg:         cfg,
		httpClient:  core.NewHTTPClient(cfg.Timeout),
		tokenClient: &http.Client{Timeout: cfg.Timeout},
	}
}

// authHeaders returns the base X-API-KEY header. Used internally by gatewayHeaders.
func (c *reimbursementHTTPClient) authHeaders() map[string]string {
	return map[string]string{
		"X-API-KEY": c.cfg.APIKey,
	}
}

// gatewayHeaders returns auth headers for all RS API calls — X-API-KEY
// plus a cached Auth0 Bearer token when M2M config is present.
func (c *reimbursementHTTPClient) gatewayHeaders(ctx context.Context) (map[string]string, error) {
	h := c.authHeaders()
	tok, err := c.m2mToken(ctx)
	if err != nil {
		return nil, err
	}
	if tok != "" {
		h["Authorization"] = "Bearer " + tok
	}
	return h, nil
}

// auth0TokenResponse is the JSON body returned by the Auth0 token endpoint.
type auth0TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

// m2mToken returns a cached-or-freshly-fetched Auth0 Bearer token obtained via
// the private-key JWT client_credentials grant. Returns "" (no error) when M2M
// config is not set — callers treat this as "no Bearer needed".
func (c *reimbursementHTTPClient) m2mToken(ctx context.Context) (string, error) {
	if c.cfg.Auth0TokenURL == "" {
		return "", nil
	}
	// Parse the private key once; subsequent calls reuse the cached result.
	c.keyOnce.Do(func() {
		c.privateKey, c.keyErr = parseRSAPrivateKey(c.cfg.Auth0ClientPrivateKey)
	})
	if c.keyErr != nil {
		return "", fmt.Errorf("m2m: parse private key: %w", c.keyErr)
	}

	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()
	if c.tokenVal != "" && time.Now().Before(c.tokenExpiry) {
		return c.tokenVal, nil
	}

	// Build JWT assertion — aud is the Auth0 tenant root derived from the token URL.
	// Per RFC 7521 the assertion aud must identify the authorisation server.
	assertionAud, err := tokenURLTenantRoot(c.cfg.Auth0TokenURL)
	if err != nil {
		return "", fmt.Errorf("m2m: derive assertion audience: %w", err)
	}
	now := time.Now()
	claims := jwt.MapClaims{
		"iss": c.cfg.Auth0ClientID,
		"sub": c.cfg.Auth0ClientID,
		"aud": assertionAud,
		"exp": now.Add(60 * time.Second).Unix(),
		"iat": now.Unix(),
	}
	signedAssertion, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(c.privateKey)
	if err != nil {
		return "", fmt.Errorf("m2m: sign assertion: %w", err)
	}

	// POST form-encoded token request using private-key JWT client assertion.
	form := url.Values{
		"grant_type":            {"client_credentials"},
		"client_assertion_type": {"urn:ietf:params:oauth:client-assertion-type:jwt-bearer"},
		"client_assertion":      {signedAssertion},
		"audience":              {c.cfg.Auth0Audience},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.Auth0TokenURL,
		strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("m2m: build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.tokenClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("m2m: token fetch: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body) // drain so the connection can be reused
		return "", fmt.Errorf("m2m: auth0 returned %d", resp.StatusCode)
	}
	var tr auth0TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return "", fmt.Errorf("m2m: decode token response: %w", err)
	}
	if tr.AccessToken == "" {
		return "", fmt.Errorf("m2m: auth0 returned empty access_token")
	}
	// Cache with a safety buffer so we never hand a near-expired token to the
	// downstream gateway. Clamp to half the TTL so short-lived tokens (≤120s)
	// don't produce a negative or zero duration.
	const bufferSec = 60
	ttl := time.Duration(tr.ExpiresIn) * time.Second
	buffer := time.Duration(bufferSec) * time.Second
	if ttl <= 2*buffer {
		buffer = ttl / 2
	}
	c.tokenVal = tr.AccessToken
	c.tokenExpiry = time.Now().Add(ttl - buffer)
	return c.tokenVal, nil
}

// parseRSAPrivateKey decodes a PEM block and parses either a PKCS8 or PKCS1
// RSA private key, returning an error if the PEM is missing or the key is not RSA.
func parseRSAPrivateKey(pemStr string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("no PEM block found in private key")
	}
	// Try PKCS8 first (the standard modern format), then fall back to PKCS1.
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		rsaKey, pkcs1Err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if pkcs1Err != nil {
			return nil, fmt.Errorf("parse private key: %w (PKCS8 err: %v)", pkcs1Err, err)
		}
		return rsaKey, nil
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key PEM is not an RSA key")
	}
	return rsaKey, nil
}

// tokenURLTenantRoot derives the Auth0 tenant root URL from the token endpoint URL,
// e.g. https://linuxfoundation-dev.auth0.com/oauth/token → https://linuxfoundation-dev.auth0.com/
// This is used as the `aud` claim in the JWT client assertion.
func tokenURLTenantRoot(tokenURL string) (string, error) {
	u, err := url.Parse(tokenURL)
	if err != nil {
		return "", fmt.Errorf("invalid token URL %q: %w", tokenURL, err)
	}
	return u.Scheme + "://" + u.Host + "/", nil
}

// rsURL builds the full endpoint URL for a given initiative ID.
func (c *reimbursementHTTPClient) rsURL(initiativeID string) string {
	return strings.TrimRight(c.cfg.APIURL, "/") + "/reimbursement/" + initiativeID
}

// SyncPolicy upserts an initiative's policy in the Reimbursement Service.
// It attempts a PATCH first (update); on 404 it falls back to POST (create),
// matching the LFF legacy behaviour. Only published initiatives are synced.
func (c *reimbursementHTTPClient) SyncPolicy(ctx context.Context, initiative *models.Initiative, ownerUser *models.User) error {
	ctx, span := reimbursementTracer.Start(ctx, "reimbursement.SyncPolicy")
	defer span.End()
	span.SetAttributes(
		attribute.String("initiative.id", initiative.ID),
		attribute.String("initiative.status", string(initiative.Status)),
	)

	// Only sync published initiatives — guard mirrors LFF PolicyCreate/UpdateFromProject.
	if !initiative.Status.EqualFold(models.StatusPublished) {
		return nil
	}

	update, create := c.buildPolicyPayload(initiative, ownerUser)

	headers, err := c.gatewayHeaders(ctx)
	if err != nil {
		return fmt.Errorf("reimbursement: fetch auth headers: %w", err)
	}
	url := c.rsURL(initiative.ID)

	// Attempt PATCH (update existing policy).
	patchErr := c.httpClient.PatchJSON(ctx, url, headers, update, nil, func(r *http.Response) error {
		return &rsHTTPError{code: r.StatusCode}
	})
	if patchErr == nil {
		return nil
	}

	// Fall back to POST (create new policy) only when the PATCH got a 404.
	var httpErr *rsHTTPError
	if !errors.As(patchErr, &httpErr) || httpErr.code != http.StatusNotFound {
		return fmt.Errorf("reimbursement: PATCH policy: %w", patchErr)
	}

	if postErr := c.httpClient.PostJSON(ctx, url, headers, create, nil, func(r *http.Response) error {
		return &rsHTTPError{code: r.StatusCode}
	}); postErr != nil {
		return fmt.Errorf("reimbursement: POST policy: %w", postErr)
	}
	return nil
}

// buildPolicyPayload builds both the PATCH and POST request bodies for an
// initiative. It consolidates the legacy buildPolicyUpdateFromProject and
// buildPolicyUpdateFromEntity functions, which handled the now-unified
// initiative domain model separately.
func (c *reimbursementHTTPClient) buildPolicyPayload(
	initiative *models.Initiative,
	ownerUser *models.User,
) (update rsPolicyUpdate, create rsPolicyCreate) {
	iType := strings.ToLower(initiative.InitiativeType)

	// --- Owner -----------------------------------------------------------
	ownerEmail := ownerUser.Email
	ownerName := strings.TrimSpace(ownerUser.GivenName + " " + ownerUser.FamilyName)
	if ownerName == "" {
		ownerName = ownerUser.Name
	}

	// --- Beneficiaries ---------------------------------------------------
	beneficiaries := make([]reimbursementContact, 0, len(initiative.Beneficiaries))
	for _, b := range initiative.Beneficiaries {
		beneficiaries = append(beneficiaries, reimbursementContact{
			Name:         b.Name,
			EmailAddress: b.Email,
		})
	}

	// --- Categories ------------------------------------------------------
	// Include all goals with a non-zero budget. For ostif, general_fund, and
	// security_audit initiatives every goal is always included regardless of
	// budget, matching the legacy entity behaviour (these types have fixed
	// cost structures where every category should appear in the Expensify policy
	// even before any donations arrive).
	alwaysInclude := iType == "ostif" || iType == "general_fund" || iType == "security_audit"
	categories := make([]string, 0, len(initiative.Goals))
	for _, g := range initiative.Goals {
		if g.AmountInCents > 0 || alwaysInclude {
			categories = append(categories, categoryName(g.Name))
		}
	}

	// --- ProjectName / display name --------------------------------------
	// For entity-like initiative types (event, mentorship, general_fund) the
	// name is prefixed with a title-cased type, e.g. "Event - KVM Forum 2019",
	// mirroring the legacy buildPolicyUpdateFromEntity prefix logic.
	displayName := initiative.Name
	if !skipTypePrefix[iType] {
		displayName = titleCaseType(initiative.InitiativeType) + " - " + displayName
	}

	// --- ProjectURL ------------------------------------------------------
	base := strings.TrimRight(c.cfg.FrontendBase, "/")
	projectURL := base + "/initiatives/" + initiative.Slug

	// --- Assemble --------------------------------------------------------
	update = rsPolicyUpdate{
		Beneficiaries: beneficiaries,
		Categories:    categories,
		Owner: reimbursementContact{
			EmailAddress: ownerEmail,
			Name:         ownerName,
		},
		ProjectURL: projectURL,
		EntityType: initiative.InitiativeType,
	}
	create = rsPolicyCreate{
		rsPolicyUpdate: update,
		ProjectName:    displayName,
	}
	return update, create
}

// skipTypePrefix lists initiative types that must NOT have their type prepended
// to the display name sent to the RS. "project" matches legacy project behaviour;
// "other" and "ostif"/"security_audit" match legacy entity behaviour.
var skipTypePrefix = map[string]bool{
	"project":        true,
	"other":          true,
	"ostif":          true,
	"security_audit": true,
}

// titleCaseType converts an initiative_type snake_case string to a title-cased
// display prefix, e.g. "general_fund" → "General Fund", "event" → "Event".
func titleCaseType(t string) string {
	words := strings.FieldsFunc(t, func(r rune) bool { return r == '_' || r == ' ' })
	for i, w := range words {
		if w != "" {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

// categoryName converts a DB goal name (lowercase, underscore-separated) to the
// PascalCase category string expected by the Reimbursement Service, e.g.
// "development" → "Development", "bug_bounty" → "BugBounty".
// The legacy "initiative" goal name is mapped to "General Fund".
func categoryName(name string) string {
	if strings.ToLower(name) == "initiative" {
		return "General Fund"
	}
	words := strings.FieldsFunc(name, func(r rune) bool { return r == '_' || r == '-' || r == ' ' })
	for i, w := range words {
		if w != "" {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, "")
}

// ProcessExpenseAction submits an action (e.g. "approve", "reject") for the
// given expense report via POST /expense/{action}/{reportId} on the
// Reimbursement Service. Authenticated with X-API-KEY and a cached Auth0
// client_credentials Bearer token (required by the API gateway).
// A 404 response is translated to domain.ErrExpenseReportNotFound.
func (c *reimbursementHTTPClient) ProcessExpenseAction(ctx context.Context, action, reportID string) error {
	ctx, span := reimbursementTracer.Start(ctx, "reimbursement.ProcessExpenseAction")
	defer span.End()
	span.SetAttributes(
		attribute.String("expense.action", action),
		attribute.String("expense.report_id", reportID),
	)

	endpoint := strings.TrimRight(c.cfg.APIURL, "/") +
		"/expense/" + url.PathEscape(action) + "/" + url.PathEscape(reportID)

	headers, err := c.gatewayHeaders(ctx)
	if err != nil {
		return fmt.Errorf("%w: fetch auth headers for %q on %s: %w", domain.ErrUpstreamUnavailable, action, reportID, err)
	}

	err = c.httpClient.PostJSON(ctx, endpoint, headers, struct{}{}, nil, func(r *http.Response) error {
		if r.StatusCode == http.StatusNotFound {
			return fmt.Errorf("%w: %s", domain.ErrExpenseReportNotFound, reportID)
		}
		return &rsHTTPError{code: r.StatusCode}
	})
	if err != nil {
		var httpErr *rsHTTPError
		if errors.As(err, &httpErr) {
			return fmt.Errorf("%w: expense action %q on %s returned %d", domain.ErrUpstreamUnavailable, action, reportID, httpErr.code)
		}
		// Network / request-build failures are also upstream outages.
		return fmt.Errorf("%w: expense action %q on %s: %w", domain.ErrUpstreamUnavailable, action, reportID, err)
	}
	return nil
}
