// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package clients provides outbound HTTP clients for external services.
package clients

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/core"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var ledgerTracer = otel.Tracer("ledger-client")

// LedgerSubTotal holds the per-category credit/debit breakdown from the Ledger
// GET /api/balance/{id} response. Used to populate donated_cents/spent_cents on goals.
type LedgerSubTotal struct {
	Credit int64 // total donated to this category (positive)
	Debit  int64 // total spent from this category (negative)
}

// LedgerBalance is the per-initiative balance returned by the Ledger service.
// SubTotals maps txnCategory strings to their credit/debit breakdown.
type LedgerBalance struct {
	InitiativeID        string
	TotalRaisedCents    int64
	TotalDisbursedCents int64
	AvailableCents      int64
	SubTotals           map[string]*LedgerSubTotal // keyed by txnCategory as returned by Ledger
}

// TransactionFilter holds query parameters for the Ledger paginate endpoint.
type TransactionFilter struct {
	ProjectID string
	TxnType   string // "donation" | "reimbursement" — empty = all
	Size      int    // page size; 0 defaults to 10
	From      int    // offset for pagination
}

// LedgerClient is the interface consumed by the service layer and the
// ledger-stats-sync CronJob.
type LedgerClient interface {
	// GetBalance fetches the current balance for a single initiative.
	GetBalance(ctx context.Context, initiativeID string) (*LedgerBalance, error)

	// GetAllBalances fetches the full bulk balance snapshot from the Ledger
	// service in one HTTP call.  Used exclusively by ledger-stats-sync.
	GetAllBalances(ctx context.Context) ([]models.LedgerRawBalance, error)

	// GetTransactions returns a paginated list of transactions for an initiative
	// from the Ledger service's Elasticsearch-backed paginate endpoint.
	GetTransactions(ctx context.Context, filter TransactionFilter) (*models.TransactionList, error)
}

// LedgerConfig holds Ledger service connection settings.
type LedgerConfig struct {
	BaseURL string
	APIKey  string
	Timeout time.Duration
}

type ledgerHTTPClient struct {
	baseURL    string // trailing slash stripped in constructor
	apiKey     string
	httpClient *core.HTTPClient
}

// NewLedgerClient creates a Ledger HTTP client with OTel-traced transport.
func NewLedgerClient(cfg LedgerConfig) LedgerClient {
	return &ledgerHTTPClient{
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:     cfg.APIKey,
		httpClient: core.NewHTTPClient(cfg.Timeout),
	}
}

type ledgerSubTotalRaw struct {
	Credit int64 `json:"credit"`
	Debit  int64 `json:"debit"`
}

type ledgerBalanceResponse struct {
	TotalRaisedCents    int64                          `json:"totalCredit"`
	TotalDisbursedCents int64                          `json:"totalDebit"`
	AvailableCents      int64                          `json:"availableBalance"`
	SubTotals           map[string]*ledgerSubTotalRaw  `json:"subTotals"`
}

// GetBalance fetches the current balance for an initiative from the Ledger service.
func (c *ledgerHTTPClient) GetBalance(ctx context.Context, initiativeID string) (*LedgerBalance, error) {
	ctx, span := ledgerTracer.Start(ctx, "ledger.GetBalance")
	defer span.End()
	span.SetAttributes(attribute.String("ledger.initiative_id", initiativeID))

	endpoint := fmt.Sprintf("%s/balance/%s", c.baseURL, initiativeID)
	headers := map[string]string{"Authorization": c.apiKey}

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
	disbursed := resp.TotalDisbursedCents
	if disbursed < 0 {
		disbursed = -disbursed
	}
	subTotals := make(map[string]*LedgerSubTotal, len(resp.SubTotals))
	for category, raw := range resp.SubTotals {
		if raw != nil {
			subTotals[category] = &LedgerSubTotal{Credit: raw.Credit, Debit: raw.Debit}
		}
	}
	return &LedgerBalance{
		InitiativeID:        initiativeID,
		TotalRaisedCents:    resp.TotalRaisedCents,
		TotalDisbursedCents: disbursed,
		AvailableCents:      resp.AvailableCents,
		SubTotals:           subTotals,
	}, nil
}

// ledgerTransactionRaw is one row from the Ledger GET /transactions/ response (Postgres-backed).
type ledgerTransactionRaw struct {
	TxnID          string `json:"txnID"`
	ProjectID      string `json:"projectID"`
	UserID         string `json:"userID"`
	OrganizationID string `json:"organizationID"`
	AccountEmail   string `json:"accountEmail"`
	SubmitterName  string `json:"submitterName"`
	TxnType        string `json:"txnType"`      // "credit" | "debit"
	TxnCategory    string `json:"txnCategory"`
	Amount         int64  `json:"amount"`       // cents
	TxnDate        int64  `json:"txnDate"`      // unix seconds
}

type ledgerTransactionsResponse struct {
	TransactionsPerPage int                    `json:"transactionsPerPage"`
	CurrentPage         int                    `json:"currentPage"`
	HasNext             bool                   `json:"hasNext"`
	Transactions        []ledgerTransactionRaw `json:"transactions"`
}

// GetTransactions fetches a paginated list of transactions for an initiative
// from the Ledger service's Postgres-backed GET /transactions/ endpoint.
// startDate=0 retrieves all-time transactions.
// Page is 1-based; filter.From is converted to page number using filter.Size.
func (c *ledgerHTTPClient) GetTransactions(ctx context.Context, filter TransactionFilter) (*models.TransactionList, error) {
	ctx, span := ledgerTracer.Start(ctx, "ledger.GetTransactions")
	defer span.End()
	span.SetAttributes(attribute.String("ledger.project_id", filter.ProjectID))

	size := filter.Size
	if size <= 0 {
		size = 10
	}
	page := 1
	if size > 0 && filter.From > 0 {
		page = filter.From/size + 1
	}

	q := url.Values{}
	q.Set("projectID", filter.ProjectID)
	q.Set("startDate", "0")
	q.Set("perPage", fmt.Sprintf("%d", size))
	q.Set("page", fmt.Sprintf("%d", page))
	if filter.TxnType != "" {
		// Ledger uses "credit"/"debit"; our API accepts "donation"/"reimbursement"
		switch filter.TxnType {
		case "donation":
			q.Set("txnType", "credit")
		case "reimbursement":
			q.Set("txnType", "debit")
		default:
			q.Set("txnType", filter.TxnType)
		}
	}

	endpoint := fmt.Sprintf("%s/transactions?%s", c.baseURL, q.Encode())
	headers := map[string]string{"Authorization": c.apiKey}

	var resp ledgerTransactionsResponse
	err := c.httpClient.GetJSON(ctx, endpoint, headers, &resp, func(r *http.Response) error {
		return fmt.Errorf("ledger transactions returned %d: %w", r.StatusCode, domain.ErrUpstreamUnavailable)
	})
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("ledger transactions: %w", err)
	}

	txns := make([]models.Transaction, 0, len(resp.Transactions))
	for _, raw := range resp.Transactions {
		txnType := "donation"
		if raw.TxnType == "debit" {
			txnType = "reimbursement"
		}
		// Best-effort display name from Ledger data; will be overwritten by CF DB lookup in service.
		donorName := raw.SubmitterName
		if donorName == "" {
			donorName = raw.AccountEmail
		}
		donorType := "individual"
		if raw.OrganizationID != "" {
			donorType = "organization"
		}
		txns = append(txns, models.Transaction{
			ID:           raw.TxnID,
			Type:         txnType,
			AmountCents:  raw.Amount,
			Date:         time.Unix(raw.TxnDate, 0).UTC(),
			Category:     raw.TxnCategory,
			DonorName:    donorName,
			DonorType:    donorType,
			LedgerUserID: raw.UserID,
			LedgerOrgID:  raw.OrganizationID,
		})
	}

	// Ledger doesn't return a total count on this endpoint; use HasNext to signal more pages.
	totalCount := filter.From + len(txns)
	if resp.HasNext {
		totalCount += size // at least one more page
	}

	return &models.TransactionList{
		Data:       txns,
		TotalCount: totalCount,
		From:       filter.From,
		Size:       size,
	}, nil
}

// GetAllBalances fetches the full bulk balance snapshot from the Ledger service.
// The endpoint is GET /balance (no initiative ID suffix).
// Returns one LedgerRawBalance per project tracked in the Ledger DB.
func (c *ledgerHTTPClient) GetAllBalances(ctx context.Context) ([]models.LedgerRawBalance, error) {
	ctx, span := ledgerTracer.Start(ctx, "ledger.GetAllBalances")
	defer span.End()

	endpoint := fmt.Sprintf("%s/balance", c.baseURL)
	headers := map[string]string{"Authorization": c.apiKey}

	var resp models.LedgerAllBalances
	err := c.httpClient.GetJSON(ctx, endpoint, headers, &resp, func(r *http.Response) error {
		return fmt.Errorf("ledger GET /balance returned %d: %w", r.StatusCode, domain.ErrUpstreamUnavailable)
	})
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("ledger all balances: %w", err)
	}
	return resp.Balances, nil
}
