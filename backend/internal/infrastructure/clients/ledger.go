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
	Limit     int    // page size; 0 defaults to 10
	Offset    int    // number of records to skip; negative treated as 0
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

	// GetPlatformBalance returns platform-wide aggregate data including category
	// totals, donor split, and top sponsors from the Ledger service.
	// topLimit controls how many top organizations and individuals are returned;
	// callers are responsible for supplying a sensible value (handler defaults to 10).
	GetPlatformBalance(ctx context.Context, topLimit int) (*LedgerPlatformBalance, error)

	// GetPlatformMonthly returns monthly donation buckets for the last N months.
	GetPlatformMonthly(ctx context.Context, months int) (*LedgerPlatformMonthly, error)

	// GetPlatformRecentDonations returns the most recent platform-wide credit transactions.
	GetPlatformRecentDonations(ctx context.Context) ([]LedgerRecentDonation, error)

	// PostTransaction records a completed charge in the Ledger service.
	// Passing version="v2" instructs the Ledger service to skip its own email
	// notifications because this service handles them.
	PostTransaction(ctx context.Context, txn LedgerTransaction) error
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
	TotalRaisedCents    int64                         `json:"totalCredit"`
	TotalDisbursedCents int64                         `json:"totalDebit"`
	AvailableCents      int64                         `json:"availableBalance"`
	SubTotals           map[string]*ledgerSubTotalRaw `json:"subTotals"`
}

// GetBalance fetches the current balance for an initiative from the Ledger service.
func (c *ledgerHTTPClient) GetBalance(ctx context.Context, initiativeID string) (*LedgerBalance, error) {
	ctx, span := ledgerTracer.Start(ctx, "ledger.GetBalance")
	defer span.End()
	span.SetAttributes(attribute.String("ledger.initiative_id", initiativeID))

	endpoint := fmt.Sprintf("%s/balance/%s", c.baseURL, initiativeID)
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
	TxnType        string `json:"txnType"` // "credit" | "debit"
	TxnCategory    string `json:"txnCategory"`
	Amount         int64  `json:"amount"`  // cents
	TxnDate        int64  `json:"txnDate"` // unix seconds
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
//
// Pagination constraint: the Ledger API is page-based (1-based). offset is
// converted to a page number via offset/limit + 1, which is only exact when
// offset is a multiple of limit. In practice all callers use limit-aligned
// offsets (0, limit, 2*limit, ...) so this is not an issue. True arbitrary
// offset semantics would require fetching across page boundaries.
func (c *ledgerHTTPClient) GetTransactions(ctx context.Context, filter TransactionFilter) (*models.TransactionList, error) {
	ctx, span := ledgerTracer.Start(ctx, "ledger.GetTransactions")
	defer span.End()
	span.SetAttributes(attribute.String("ledger.project_id", filter.ProjectID))

	limit := filter.Limit
	if limit <= 0 {
		limit = 10
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}
	// Convert offset/limit to the Ledger's 1-based page model.
	page := offset/limit + 1

	q := url.Values{}
	q.Set("projectID", filter.ProjectID)
	q.Set("startDate", "0")
	q.Set("perPage", fmt.Sprintf("%d", limit))
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
	headers := map[string]string{"Authorization": "Bearer " + c.apiKey}

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
			DonorType:    donorType,
			DonorName:    raw.SubmitterName,
			LedgerUserID: raw.UserID,
			LedgerOrgID:  raw.OrganizationID,
		})
	}

	// Ledger doesn't return a total count on this endpoint; use HasNext to estimate.
	totalCount := offset + len(txns)
	if resp.HasNext {
		totalCount += limit // at least one more page
	}

	return &models.TransactionList{
		Data:       txns,
		TotalCount: totalCount,
		Limit:      limit,
		Offset:     offset,
	}, nil
}

// LedgerPlatformBalance holds the platform-wide aggregate returned by GET /balance/platform.
type LedgerPlatformBalance struct {
	TotalSupporters    int64
	OrganizationsCents int64
	IndividualsCents   int64
	Categories         []LedgerCategoryTotal
	TopOrganizations   []LedgerSponsorRaw
	TopIndividuals     []LedgerSponsorRaw
}

// LedgerCategoryTotal is one category entry from the platform balance response.
type LedgerCategoryTotal struct {
	Name       string
	TotalCents int64
	Count      int
}

// LedgerSponsorRaw is a top-sponsor entry from the platform balance response.
type LedgerSponsorRaw struct {
	ID    string
	Total int64
}

// LedgerPlatformMonthly holds monthly donation buckets from GET /balance/platform/monthly.
type LedgerPlatformMonthly struct {
	Buckets []LedgerMonthlyBucket
}

// LedgerMonthlyBucket is one calendar-month entry from the platform monthly response.
type LedgerMonthlyBucket struct {
	Year          int
	Month         int
	TotalCents    int64
	Supporters    int64
	NewSupporters int64
}

// LedgerRecentDonation is one entry from GET /transactions/platform/recent.
type LedgerRecentDonation struct {
	TxnID          string
	ProjectID      string
	UserID         string
	OrganizationID string
	SubmitterName  string
	Amount         int64
	TxnDate        int64
	TxnCategory    string
	SourceType     string
}

// raw JSON types for platform balance response
type ledgerPlatformBalanceResponse struct {
	TotalSupporters    int64 `json:"totalSupporters"`
	OrganizationsCents int64 `json:"organizationsCents"`
	IndividualsCents   int64 `json:"individualsCents"`
	Categories         []struct {
		Name       string `json:"name"`
		TotalCents int64  `json:"totalCents"`
		Count      int    `json:"count"`
	} `json:"categories"`
	TopOrganizations []struct {
		ID    string `json:"id"`
		Total int64  `json:"total"`
	} `json:"topOrganizations"`
	TopIndividuals []struct {
		ID    string `json:"id"`
		Total int64  `json:"total"`
	} `json:"topIndividuals"`
}

// raw JSON types for platform monthly response
type ledgerPlatformMonthlyResponse struct {
	Buckets []struct {
		Year          int   `json:"year"`
		Month         int   `json:"month"`
		TotalCents    int64 `json:"totalCents"`
		Supporters    int64 `json:"supporters"`
		NewSupporters int64 `json:"newSupporters"`
	} `json:"buckets"`
}

// raw JSON types for platform recent transactions response
type ledgerPlatformRecentResponse struct {
	Data []struct {
		TxnID          string `json:"txnID"`
		ProjectID      string `json:"projectID"`
		UserID         string `json:"userID"`
		OrganizationID string `json:"organizationID"`
		SubmitterName  string `json:"submitterName"`
		Amount         int64  `json:"amount"`
		TxnDate        int64  `json:"txnDate"`
		TxnCategory    string `json:"txnCategory"`
		SourceType     string `json:"sourceType"`
	} `json:"data"`
}

// GetPlatformBalance fetches platform-wide aggregate data from the Ledger service.
func (c *ledgerHTTPClient) GetPlatformBalance(ctx context.Context, topLimit int) (*LedgerPlatformBalance, error) {
	ctx, span := ledgerTracer.Start(ctx, "ledger.GetPlatformBalance")
	defer span.End()

	if topLimit <= 0 {
		topLimit = 10
	}
	endpoint := fmt.Sprintf("%s/balance/platform?top_limit=%d", c.baseURL, topLimit)
	headers := map[string]string{"Authorization": "Bearer " + c.apiKey}

	var resp ledgerPlatformBalanceResponse
	err := c.httpClient.GetJSON(ctx, endpoint, headers, &resp, func(r *http.Response) error {
		return fmt.Errorf("ledger GET /balance/platform returned %d: %w", r.StatusCode, domain.ErrUpstreamUnavailable)
	})
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("ledger platform balance: %w", err)
	}

	out := &LedgerPlatformBalance{
		TotalSupporters:    resp.TotalSupporters,
		OrganizationsCents: resp.OrganizationsCents,
		IndividualsCents:   resp.IndividualsCents,
	}
	for _, c := range resp.Categories {
		out.Categories = append(out.Categories, LedgerCategoryTotal{Name: c.Name, TotalCents: c.TotalCents, Count: c.Count})
	}
	for _, o := range resp.TopOrganizations {
		out.TopOrganizations = append(out.TopOrganizations, LedgerSponsorRaw{ID: o.ID, Total: o.Total})
	}
	for _, i := range resp.TopIndividuals {
		out.TopIndividuals = append(out.TopIndividuals, LedgerSponsorRaw{ID: i.ID, Total: i.Total})
	}
	return out, nil
}

// GetPlatformMonthly fetches monthly donation buckets for the last N months.
func (c *ledgerHTTPClient) GetPlatformMonthly(ctx context.Context, months int) (*LedgerPlatformMonthly, error) {
	ctx, span := ledgerTracer.Start(ctx, "ledger.GetPlatformMonthly")
	defer span.End()
	span.SetAttributes(attribute.Int("ledger.months", months))

	endpoint := fmt.Sprintf("%s/balance/platform/monthly?months=%d", c.baseURL, months)
	headers := map[string]string{"Authorization": "Bearer " + c.apiKey}

	var resp ledgerPlatformMonthlyResponse
	err := c.httpClient.GetJSON(ctx, endpoint, headers, &resp, func(r *http.Response) error {
		return fmt.Errorf("ledger GET /balance/platform/monthly returned %d: %w", r.StatusCode, domain.ErrUpstreamUnavailable)
	})
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("ledger platform monthly: %w", err)
	}

	out := &LedgerPlatformMonthly{}
	for _, b := range resp.Buckets {
		out.Buckets = append(out.Buckets, LedgerMonthlyBucket{
			Year:          b.Year,
			Month:         b.Month,
			TotalCents:    b.TotalCents,
			Supporters:    b.Supporters,
			NewSupporters: b.NewSupporters,
		})
	}
	return out, nil
}

// GetPlatformRecentDonations fetches the most recent platform-wide credit transactions.
func (c *ledgerHTTPClient) GetPlatformRecentDonations(ctx context.Context) ([]LedgerRecentDonation, error) {
	ctx, span := ledgerTracer.Start(ctx, "ledger.GetPlatformRecentDonations")
	defer span.End()

	endpoint := fmt.Sprintf("%s/transactions/platform/recent", c.baseURL)
	headers := map[string]string{"Authorization": "Bearer " + c.apiKey}

	var resp ledgerPlatformRecentResponse
	err := c.httpClient.GetJSON(ctx, endpoint, headers, &resp, func(r *http.Response) error {
		return fmt.Errorf("ledger GET /transactions/platform/recent returned %d: %w", r.StatusCode, domain.ErrUpstreamUnavailable)
	})
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("ledger platform recent donations: %w", err)
	}

	out := make([]LedgerRecentDonation, 0, len(resp.Data))
	for _, d := range resp.Data {
		out = append(out, LedgerRecentDonation{
			TxnID:          d.TxnID,
			ProjectID:      d.ProjectID,
			UserID:         d.UserID,
			OrganizationID: d.OrganizationID,
			SubmitterName:  d.SubmitterName,
			Amount:         d.Amount,
			TxnDate:        d.TxnDate,
			TxnCategory:    d.TxnCategory,
			SourceType:     d.SourceType,
		})
	}
	return out, nil
}

// GetAllBalances fetches the full bulk balance snapshot from the Ledger service.
// The endpoint is GET /balance (no initiative ID suffix).
// Returns one LedgerRawBalance per project tracked in the Ledger DB.
func (c *ledgerHTTPClient) GetAllBalances(ctx context.Context) ([]models.LedgerRawBalance, error) {
	ctx, span := ledgerTracer.Start(ctx, "ledger.GetAllBalances")
	defer span.End()

	endpoint := fmt.Sprintf("%s/balance", c.baseURL)
	headers := map[string]string{"Authorization": "Bearer " + c.apiKey}

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

// LedgerTransaction is the wire representation of a single transaction sent to
// the Ledger POST /transactions endpoint. Field names match the Ledger service's
// JSON schema exactly (camelCase).
type LedgerTransaction struct {
	ProjectID       string `json:"projectID"`
	UserID          string `json:"userID"`
	OrganizationID  string `json:"organizationID,omitempty"`
	AccountEmail    string `json:"accountEmail"`
	TxnComment      string `json:"txnComment,omitempty"`
	SourceType      string `json:"sourceType"`
	SourceTxnID     string `json:"sourceTxnID"`
	SourceAccountID string `json:"sourceAccountID"`
	TxnType         string `json:"txnType"`
	TxnCategory     string `json:"txnCategory,omitempty"`
	Fee             int    `json:"fee"`
	Amount          int    `json:"amount"`
	TxnDate         int64  `json:"txnDate"`
}

type ledgerPostTransactionsRequest struct {
	Transactions []LedgerTransaction `json:"transactions"`
}

// PostTransaction records a completed charge in the Ledger service.
// The request is sent with ?version=v2 so the Ledger service skips its own
// email notifications — this service is responsible for donor/admin emails.
func (c *ledgerHTTPClient) PostTransaction(ctx context.Context, txn LedgerTransaction) error {
	ctx, span := ledgerTracer.Start(ctx, "ledger.PostTransaction")
	defer span.End()
	span.SetAttributes(
		attribute.String("ledger.project_id", txn.ProjectID),
		attribute.String("ledger.source_txn_id", txn.SourceTxnID),
	)

	endpoint := fmt.Sprintf("%s/transactions?version=v2", c.baseURL)
	headers := map[string]string{"Authorization": "Bearer " + c.apiKey}
	body := ledgerPostTransactionsRequest{Transactions: []LedgerTransaction{txn}}

	if err := c.httpClient.PostJSON(ctx, endpoint, headers, body, nil, func(r *http.Response) error {
		return fmt.Errorf("ledger POST /transactions returned %d: %w", r.StatusCode, domain.ErrUpstreamUnavailable)
	}); err != nil {
		span.RecordError(err)
		return fmt.Errorf("ledger post transaction: %w", err)
	}
	return nil
}
