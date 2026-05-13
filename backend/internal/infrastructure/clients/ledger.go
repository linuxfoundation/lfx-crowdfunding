// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package clients provides outbound HTTP clients for external services.
package clients

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/core"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var ledgerTracer = otel.Tracer("ledger-client")

// LedgerClient is the interface consumed by the service layer.
// Balance and transaction data are NEVER stored in PostgreSQL — always fetched live.
type LedgerClient interface {
	GetBalance(ctx context.Context, initiativeID string) (*models.Balance, error)
}

// LedgerConfig holds Ledger service connection settings.
type LedgerConfig struct {
	BaseURL string
	APIKey  string
	Timeout time.Duration
}

type ledgerHTTPClient struct {
	baseURL    string
	apiKey     string
	httpClient *core.HTTPClient
}

// NewLedgerClient creates a Ledger HTTP client with OTel-traced transport.
func NewLedgerClient(cfg LedgerConfig) LedgerClient {
	return &ledgerHTTPClient{
		baseURL:    cfg.BaseURL,
		apiKey:     cfg.APIKey,
		httpClient: core.NewHTTPClient(cfg.Timeout),
	}
}

type ledgerBalanceResponse struct {
	InitiativeID        string `json:"projectId"`
	TotalRaisedCents    int64  `json:"totalRaisedCents"`
	TotalDisbursedCents int64  `json:"totalDisbursedCents"`
	AvailableCents      int64  `json:"availableCents"`
}

// GetBalance fetches the current balance for an initiative from the Ledger service.
func (c *ledgerHTTPClient) GetBalance(ctx context.Context, initiativeID string) (*models.Balance, error) {
	ctx, span := ledgerTracer.Start(ctx, "ledger.GetBalance")
	defer span.End()
	span.SetAttributes(attribute.String("ledger.initiative_id", initiativeID))

	endpoint := fmt.Sprintf("%s/v1/projects/%s/balance", c.baseURL, initiativeID)
	headers := map[string]string{"Authorization": "Bearer " + c.apiKey}

	var resp ledgerBalanceResponse
	err := c.httpClient.GetJSON(ctx, endpoint, headers, &resp, func(r *http.Response) error {
		if r.StatusCode == http.StatusNotFound {
			return domain.ErrInitiativeNotFound
		}
		return domain.ErrUpstreamUnavailable
	})
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("ledger balance: %w", err)
	}
	return &models.Balance{
		InitiativeID:        initiativeID,
		TotalRaisedCents:    resp.TotalRaisedCents,
		TotalDisbursedCents: resp.TotalDisbursedCents,
		AvailableCents:      resp.AvailableCents,
	}, nil
}
