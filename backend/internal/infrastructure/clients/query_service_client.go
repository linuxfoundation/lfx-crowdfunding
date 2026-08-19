// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// QueryServiceConfig holds settings for calling the v2 platform query service
// through the public API gateway.
type QueryServiceConfig struct {
	// BaseURL is the gateway base URL, e.g. https://api-gw.platform.linuxfoundation.org
	BaseURL string
	// Timeout caps a single query-service HTTP call.
	Timeout time.Duration
}

// OrgCandidate is a v2 platform organization (member-service b2b_org) the
// caller has a direct FGA relation to — surfaced to the fundraise-form
// affiliation picker.
type OrgCandidate struct {
	UID     string `json:"uid"`
	Name    string `json:"name"`
	LogoURL string `json:"logo_url"`
}

// QueryServiceClient reads resources from the v2 platform query service.
type QueryServiceClient interface {
	// SearchOrganizationsForUser returns the b2b_org resources the caller
	// (identified by the given v2-audience bearer token) has a direct FGA
	// relation to (writer or auditor). Does not expand inherited grants.
	SearchOrganizationsForUser(ctx context.Context, bearerToken string) ([]OrgCandidate, error)
}

type queryServiceHTTPClient struct {
	cfg        QueryServiceConfig
	httpClient *http.Client
}

// NewQueryServiceClient returns a QueryServiceClient, or nil when cfg.BaseURL
// is empty — callers treat a nil client as "query-service lookup disabled".
func NewQueryServiceClient(cfg QueryServiceConfig) QueryServiceClient {
	if cfg.BaseURL == "" {
		return nil
	}
	return &queryServiceHTTPClient{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: cfg.Timeout},
	}
}

// queryServiceItem mirrors QueryServiceItem<T> from the query-service
// contract: {type, id, data}. id is prefixed "b2b_org:<uid>".
type queryServiceItem struct {
	ID   string `json:"id"`
	Data struct {
		Name    string `json:"name"`
		LogoURL string `json:"logo_url"`
	} `json:"data"`
}

type queryServiceResponse struct {
	Resources []queryServiceItem `json:"resources"`
}

func (c *queryServiceHTTPClient) SearchOrganizationsForUser(ctx context.Context, bearerToken string) ([]OrgCandidate, error) {
	endpoint := strings.TrimRight(c.cfg.BaseURL, "/") + "/query/resources?v=1&type=b2b_org&filter_grants=direct"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("query service: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+bearerToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("query service: request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body) // drain so the connection can be reused
		return nil, fmt.Errorf("query service: returned %d", resp.StatusCode)
	}

	var qr queryServiceResponse
	if err := json.NewDecoder(resp.Body).Decode(&qr); err != nil {
		return nil, fmt.Errorf("query service: decode response: %w", err)
	}

	orgs := make([]OrgCandidate, 0, len(qr.Resources))
	for _, item := range qr.Resources {
		orgs = append(orgs, OrgCandidate{
			UID:     extractUID(item.ID),
			Name:    item.Data.Name,
			LogoURL: item.Data.LogoURL,
		})
	}
	return orgs, nil
}

// extractUID strips the "<type>:" prefix query-service prepends to
// resource.id (e.g. "b2b_org:0012M00002qnukOQAQ" -> "0012M00002qnukOQAQ").
func extractUID(resourceID string) string {
	if idx := strings.IndexByte(resourceID, ':'); idx != -1 {
		return resourceID[idx+1:]
	}
	return resourceID
}
