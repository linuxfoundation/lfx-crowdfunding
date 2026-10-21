// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package clients

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/core"
	"go.opentelemetry.io/otel"
)

var projectLookupTracer = otel.Tracer("project-lookup-client")

// ProjectLookup answers "may an initiative be attributed to this project?"
// (lfx-crowdfunding#336, Doc 11 §2.1): only public projects past formation
// are. It calls the v2 query service through the LFX API gateway anonymously;
// the gateway's anonymous authenticator only returns public project docs, so
// a private or nonexistent project comes back as no result.
type ProjectLookup struct {
	baseURL    string
	httpClient *core.HTTPClient
}

// NewProjectLookup returns a ProjectLookup for the given gateway base URL
// (e.g. https://lfx-api.dev.v2.cluster.linuxfound.info).
func NewProjectLookup(baseURL string, timeout time.Duration) *ProjectLookup {
	return &ProjectLookup{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: core.NewHTTPClient(timeout),
	}
}

type projectDoc struct {
	Public bool   `json:"public"`
	Stage  string `json:"stage"`
}

type projectQueryResponse struct {
	Resources []struct {
		ID   string     `json:"id"`
		Data projectDoc `json:"data"`
	} `json:"resources"`
}

// IsAttributable reports whether uid is a public project that is not in
// formation. A miss (private, nonexistent) returns (false, nil); any upstream
// failure wraps domain.ErrUpstreamUnavailable and is never turned into false.
func (c *ProjectLookup) IsAttributable(ctx context.Context, uid string) (bool, error) {
	ctx, span := projectLookupTracer.Start(ctx, "project-lookup.IsAttributable")
	defer span.End()

	q := url.Values{"v": {"1"}, "type": {"project"}, "filters": {"uid:" + uid}}
	var resp projectQueryResponse
	err := c.httpClient.GetJSON(ctx, c.baseURL+"/query/resources?"+q.Encode(), nil, &resp, func(r *http.Response) error {
		return fmt.Errorf("%w: query service returned %d", domain.ErrUpstreamUnavailable, r.StatusCode)
	})
	if err != nil {
		span.RecordError(err)
		if !errors.Is(err, domain.ErrUpstreamUnavailable) {
			err = fmt.Errorf("%w: %w", domain.ErrUpstreamUnavailable, err)
		}
		return false, fmt.Errorf("project lookup: %w", err)
	}

	for _, r := range resp.Resources {
		if r.ID == "project:"+uid {
			return r.Data.Public && !inFormation(r.Data.Stage), nil
		}
	}
	return false, nil
}

// inFormation reports whether a project-service stage has no public identity
// yet: any "Formation - …" stage, or "Prospect". A missing stage passes.
func inFormation(stage string) bool {
	return stage == "Prospect" || strings.HasPrefix(stage, "Formation - ")
}
