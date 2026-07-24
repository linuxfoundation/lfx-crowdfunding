// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/clients"
)

// capturingLedger records the last TransactionFilter forwarded to GetTransactions
// and returns the configured response. All other LedgerClient methods are no-ops.
type capturingLedger struct {
	lastFilter clients.TransactionFilter
	resp       *models.TransactionList
	err        error
}

func (c *capturingLedger) GetBalance(_ context.Context, _ string) (*clients.LedgerBalance, error) {
	return nil, nil
}
func (c *capturingLedger) GetAllBalances(_ context.Context) ([]models.LedgerRawBalance, error) {
	return nil, nil
}
func (c *capturingLedger) GetTransactions(_ context.Context, f clients.TransactionFilter) (*models.TransactionList, error) {
	c.lastFilter = f
	if c.err != nil {
		return nil, c.err
	}
	if c.resp != nil {
		return c.resp, nil
	}
	return &models.TransactionList{}, nil
}
func (c *capturingLedger) GetPlatformBalance(_ context.Context, _ int) (*clients.LedgerPlatformBalance, error) {
	return nil, nil
}
func (c *capturingLedger) GetPlatformMonthly(_ context.Context, _ int) (*clients.LedgerPlatformMonthly, error) {
	return nil, nil
}
func (c *capturingLedger) GetPlatformRecentDonations(_ context.Context) ([]clients.LedgerRecentDonation, error) {
	return nil, nil
}
func (c *capturingLedger) GetOrgDonations(_ context.Context) ([]clients.LedgerOrgDonation, error) {
	return nil, nil
}
func (c *capturingLedger) PostTransaction(_ context.Context, _ clients.LedgerTransaction) error {
	return nil
}

// newMyTxnSvc creates a minimal InitiativeService backed by the given Ledger stub.
// Repo enrichment methods return empty maps so transactions pass through unmodified.
func newMyTxnSvc(t *testing.T, ledger *capturingLedger) *InitiativeService {
	t.Helper()
	return NewInitiativeService(
		&mockInitiativeRepo{},
		&mockUserRepository{},
		ledger,
		&mockStripeClient{},
		&mockEmailService{},
		nil,
		slog.Default(),
	)
}

const (
	testInitiativeID = "init-1"
	testUserID       = "auth0|user1"
	testOtherUserID  = "auth0|user2"
)

// txn constructs a minimal Transaction with the given user and amount.
func txn(userID string, amountCents int64) models.Transaction {
	return models.Transaction{
		ID:           "txn-" + userID,
		Type:         "donation",
		AmountCents:  amountCents,
		Date:         time.Time{},
		LedgerUserID: userID,
	}
}

// --- Tests ---

// TestGetMyTransactions_ForwardsSubscriptionOnly verifies that SubscriptionOnly
// is set in the TransactionFilter forwarded to the Ledger client.
func TestGetMyTransactions_ForwardsSubscriptionOnly(t *testing.T) {
	ledger := &capturingLedger{
		resp: &models.TransactionList{
			Data:  []models.Transaction{txn(testUserID, 300)},
			Limit: 10,
		},
	}
	svc := newMyTxnSvc(t, ledger)

	_, err := svc.GetMyTransactions(context.Background(), testInitiativeID, testUserID, "", true, 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ledger.lastFilter.SubscriptionOnly {
		t.Error("TransactionFilter.SubscriptionOnly should be true when subscriptionOnly=true is passed")
	}
}

// TestGetMyTransactions_ForwardsUserID verifies that UserID is set in the
// TransactionFilter forwarded to the Ledger client.
func TestGetMyTransactions_ForwardsUserID(t *testing.T) {
	ledger := &capturingLedger{resp: &models.TransactionList{Limit: 10}}
	svc := newMyTxnSvc(t, ledger)

	_, err := svc.GetMyTransactions(context.Background(), testInitiativeID, testUserID, "", false, 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ledger.lastFilter.UserID != testUserID {
		t.Errorf("UserID in filter = %q, want %q", ledger.lastFilter.UserID, testUserID)
	}
	if ledger.lastFilter.ProjectID != testInitiativeID {
		t.Errorf("ProjectID in filter = %q, want %q", ledger.lastFilter.ProjectID, testInitiativeID)
	}
}

// TestGetMyTransactions_HappyPath verifies that when all returned rows belong to
// the requested user they are returned unchanged.
func TestGetMyTransactions_HappyPath(t *testing.T) {
	rows := []models.Transaction{
		txn(testUserID, 500),
		txn(testUserID, 300),
	}
	ledger := &capturingLedger{
		resp: &models.TransactionList{Data: rows, TotalCount: 2, Limit: 10},
	}
	svc := newMyTxnSvc(t, ledger)

	list, err := svc.GetMyTransactions(context.Background(), testInitiativeID, testUserID, "", false, 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list.Data) != 2 {
		t.Errorf("len(Data) = %d, want 2", len(list.Data))
	}
}

// TestGetMyTransactions_ForeignRowsError verifies that when the Ledger returns any
// row belonging to a different user (server-side filtering unavailable), the method
// returns ErrUpstreamUnavailable rather than incorrect results.
func TestGetMyTransactions_ForeignRowsError(t *testing.T) {
	rows := []models.Transaction{
		txn(testUserID, 500),
		txn(testOtherUserID, 300), // foreign row — Ledger ignored userID param
	}
	ledger := &capturingLedger{
		resp: &models.TransactionList{Data: rows, TotalCount: 20, Limit: 10},
	}
	svc := newMyTxnSvc(t, ledger)

	_, err := svc.GetMyTransactions(context.Background(), testInitiativeID, testUserID, "donation", false, 10, 0)
	if err == nil {
		t.Fatal("expected error for foreign rows, got nil")
	}
	if !errors.Is(err, domain.ErrUpstreamUnavailable) {
		t.Errorf("error = %v, want to wrap domain.ErrUpstreamUnavailable", err)
	}
}

// TestGetMyTransactions_AllForeignRowsError verifies the case described in the review:
// a page containing only foreign transactions (data:[] after filter, total_count>0).
func TestGetMyTransactions_AllForeignRowsError(t *testing.T) {
	rows := []models.Transaction{
		txn(testOtherUserID, 300),
		txn(testOtherUserID, 200),
	}
	ledger := &capturingLedger{
		resp: &models.TransactionList{Data: rows, TotalCount: 10, Limit: 2},
	}
	svc := newMyTxnSvc(t, ledger)

	_, err := svc.GetMyTransactions(context.Background(), testInitiativeID, testUserID, "donation", false, 2, 0)
	if err == nil {
		t.Fatal("expected error for all-foreign page, got nil")
	}
	if !errors.Is(err, domain.ErrUpstreamUnavailable) {
		t.Errorf("error = %v, want to wrap domain.ErrUpstreamUnavailable", err)
	}
}

// TestGetMyTransactions_ExcludesNegativeDonations verifies that negative-amount
// credit rows (grant disbursements) are excluded when type=donation, and that
// TotalCount is adjusted accordingly.
func TestGetMyTransactions_ExcludesNegativeDonations(t *testing.T) {
	rows := []models.Transaction{
		txn(testUserID, 500),
		{ID: "grant", Type: "donation", AmountCents: -1000, LedgerUserID: testUserID}, // disbursement
		txn(testUserID, 300),
	}
	// TotalCount=3, no HasNext (TotalCount == offset+len(rows))
	ledger := &capturingLedger{
		resp: &models.TransactionList{Data: rows, TotalCount: 3, Limit: 10},
	}
	svc := newMyTxnSvc(t, ledger)

	list, err := svc.GetMyTransactions(context.Background(), testInitiativeID, testUserID, "donation", false, 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list.Data) != 2 {
		t.Errorf("len(Data) = %d, want 2 (negative row removed)", len(list.Data))
	}
	// TotalCount should be decremented by the 1 dropped row.
	if list.TotalCount != 2 {
		t.Errorf("TotalCount = %d, want 2", list.TotalCount)
	}
	for _, d := range list.Data {
		if d.AmountCents <= 0 {
			t.Errorf("negative-amount row %q leaked into result (amount=%d)", d.ID, d.AmountCents)
		}
	}
}

// TestGetMyTransactions_NegativeDonationsPaginationClamp verifies that when an
// entire page of user rows is filtered out (all negative) but HasNext=true, the
// TotalCount is kept above the next page's offset so the caller continues paginating.
func TestGetMyTransactions_NegativeDonationsPaginationClamp(t *testing.T) {
	rows := []models.Transaction{
		{ID: "g1", Type: "donation", AmountCents: -500, LedgerUserID: testUserID},
		{ID: "g2", Type: "donation", AmountCents: -200, LedgerUserID: testUserID},
	}
	const (
		limit      = 2
		offset     = 0
		totalCount = 10 // > offset+len(rows), so HasNext is implied
	)
	ledger := &capturingLedger{
		resp: &models.TransactionList{Data: rows, TotalCount: totalCount, Limit: limit},
	}
	svc := newMyTxnSvc(t, ledger)

	list, err := svc.GetMyTransactions(context.Background(), testInitiativeID, testUserID, "donation", false, limit, offset)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list.Data) != 0 {
		t.Errorf("len(Data) = %d, want 0", len(list.Data))
	}
	// Must keep TotalCount > offset+limit so callers continue to page 2.
	nextOffset := offset + limit
	if list.TotalCount <= nextOffset {
		t.Errorf("TotalCount = %d must be > nextOffset %d to preserve pagination", list.TotalCount, nextOffset)
	}
}

// TestGetMyTransactions_EmptyPage verifies that an empty Ledger response (user has
// no transactions) returns successfully with no data and no error.
func TestGetMyTransactions_EmptyPage(t *testing.T) {
	ledger := &capturingLedger{resp: &models.TransactionList{Limit: 10}}
	svc := newMyTxnSvc(t, ledger)

	list, err := svc.GetMyTransactions(context.Background(), testInitiativeID, testUserID, "donation", false, 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list.Data) != 0 {
		t.Errorf("len(Data) = %d, want 0", len(list.Data))
	}
}

// ── GetAllMyTransactions ──────────────────────────────────────────────────────

// TestGetAllMyTransactions_OmitsProjectIDFilter verifies that the Ledger is
// called without a ProjectID so transactions across all initiatives are returned.
func TestGetAllMyTransactions_OmitsProjectIDFilter(t *testing.T) {
	ledger := &capturingLedger{
		resp: &models.TransactionList{
			Data:  []models.Transaction{txn(testUserID, 300)},
			Limit: 10,
		},
	}
	svc := newMyTxnSvc(t, ledger)

	_, err := svc.GetAllMyTransactions(context.Background(), testUserID, "", false, 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ledger.lastFilter.ProjectID != "" {
		t.Errorf("ProjectID in filter = %q, want empty (cross-initiative endpoint)", ledger.lastFilter.ProjectID)
	}
	if ledger.lastFilter.UserID != testUserID {
		t.Errorf("UserID in filter = %q, want %q", ledger.lastFilter.UserID, testUserID)
	}
}

// TestGetAllMyTransactions_ForeignRowsError verifies that when the Ledger returns
// any row belonging to a different user, the method returns ErrUpstreamUnavailable.
func TestGetAllMyTransactions_ForeignRowsError(t *testing.T) {
	rows := []models.Transaction{
		txn(testUserID, 500),
		txn(testOtherUserID, 300), // foreign row — Ledger ignored userID param
	}
	ledger := &capturingLedger{
		resp: &models.TransactionList{Data: rows, TotalCount: 20, Limit: 10},
	}
	svc := newMyTxnSvc(t, ledger)

	_, err := svc.GetAllMyTransactions(context.Background(), testUserID, "donation", false, 10, 0)
	if err == nil {
		t.Fatal("expected error for foreign rows, got nil")
	}
	if !errors.Is(err, domain.ErrUpstreamUnavailable) {
		t.Errorf("error = %v, want to wrap domain.ErrUpstreamUnavailable", err)
	}
}

// TestGetAllMyTransactions_ExcludesNegativeDonations verifies that negative-amount
// credit rows (grant disbursements) are excluded when type=donation, matching the
// behavior of the per-initiative GetTransactions method.
func TestGetAllMyTransactions_ExcludesNegativeDonations(t *testing.T) {
	rows := []models.Transaction{
		txn(testUserID, 500),
		{ID: "grant", Type: "donation", AmountCents: -1000, LedgerUserID: testUserID},
		txn(testUserID, 300),
	}
	ledger := &capturingLedger{
		resp: &models.TransactionList{Data: rows, TotalCount: 3, Limit: 10},
	}
	svc := newMyTxnSvc(t, ledger)

	list, err := svc.GetAllMyTransactions(context.Background(), testUserID, "donation", false, 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list.Data) != 2 {
		t.Errorf("len(Data) = %d, want 2 (negative row removed)", len(list.Data))
	}
	if list.TotalCount != 2 {
		t.Errorf("TotalCount = %d, want 2 (adjusted by dropped row)", list.TotalCount)
	}
	for _, d := range list.Data {
		if d.AmountCents <= 0 {
			t.Errorf("negative-amount row %q leaked into result (amount=%d)", d.ID, d.AmountCents)
		}
	}
}

// TestGetAllMyTransactions_EnrichesInitiativeNames verifies that when Ledger rows
// carry distinct LedgerProjectIDs, the per-item InitiativeName is populated from
// a single batch call to GetInitiativesByIDs (no N+1).
func TestGetAllMyTransactions_EnrichesInitiativeNames(t *testing.T) {
	const (
		proj1 = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
		proj2 = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	)

	rows := []models.Transaction{
		{ID: "t1", AmountCents: 100, LedgerUserID: testUserID, LedgerProjectID: proj1},
		{ID: "t2", AmountCents: 200, LedgerUserID: testUserID, LedgerProjectID: proj2},
		{ID: "t3", AmountCents: 300, LedgerUserID: testUserID, LedgerProjectID: proj1},
	}
	ledger := &capturingLedger{
		resp: &models.TransactionList{Data: rows, TotalCount: 3, Limit: 10},
	}

	initiativeMap := map[string]*models.Initiative{
		proj1: {ID: proj1, Name: "Alpha Project"},
		proj2: {ID: proj2, Name: "Beta Fund"},
	}
	batchCallCount := 0
	repo := &mockInitiativeRepo{
		onGetInitiativesByIDs: func(_ context.Context, ids []string) (map[string]*models.Initiative, error) {
			batchCallCount++
			result := make(map[string]*models.Initiative, len(ids))
			for _, id := range ids {
				if ini, ok := initiativeMap[id]; ok {
					result[id] = ini
				}
			}
			return result, nil
		},
	}

	svc := NewInitiativeService(repo, &mockUserRepository{}, ledger, &mockStripeClient{}, &mockEmailService{}, nil, slog.Default())

	list, err := svc.GetAllMyTransactions(context.Background(), testUserID, "", false, 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Exactly one batch DB call regardless of the number of distinct project IDs.
	if batchCallCount != 1 {
		t.Errorf("GetInitiativesByIDs called %d time(s), want exactly 1 (no N+1)", batchCallCount)
	}

	for _, txnItem := range list.Data {
		want := initiativeMap[txnItem.LedgerProjectID].Name
		if txnItem.InitiativeName != want {
			t.Errorf("txn %q: InitiativeName = %q, want %q", txnItem.ID, txnItem.InitiativeName, want)
		}
	}
}
