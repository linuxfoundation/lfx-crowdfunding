// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

// ledgerSource is the interface the Syncer needs from the Ledger HTTP client.
// Defined at the point of consumption per constitution Principle III.
type ledgerSource interface {
	GetAllBalances(ctx context.Context) ([]models.LedgerRawBalance, error)
}

// syncResult carries the per-run counters logged on completion.
type syncResult struct {
	total    int // initiatives in CF DB eligible for sync
	matched  int // initiatives that had a Ledger entry
	upserted int // rows successfully written to initiative_ledger_stats
	skipped  int // initiatives with no matching Ledger entry
}

// Syncer orchestrates a single ledger-stats-sync run.
type Syncer struct {
	repo   domain.LedgerStatsRepository
	ledger ledgerSource
	logger *slog.Logger
}

// newSyncer returns a configured Syncer ready to call Run.
func newSyncer(repo domain.LedgerStatsRepository, ledger ledgerSource, logger *slog.Logger) *Syncer {
	return &Syncer{repo: repo, ledger: ledger, logger: logger}
}

// Run executes the full sync algorithm:
//  1. Load all non-archived/non-draft initiative IDs from CF DB.
//  2. Fetch the full bulk balance snapshot from the Ledger service.
//  3. Build an O(1) lookup index keyed by projectID.
//  4. Collect unique org/user sponsor IDs for bulk enrichment.
//  5. Enrich sponsor entries with names and avatar/logo URLs from CF DB.
//  6. Upsert matching rows into initiative_ledger_stats.
//  7. Return per-run counters for summary logging.
func (s *Syncer) Run(ctx context.Context) (syncResult, error) {
	// Step 1 — load initiative IDs
	initiativeIDs, err := s.repo.ListActiveSyncIDs(ctx)
	if err != nil {
		return syncResult{}, fmt.Errorf("list active initiative IDs: %w", err)
	}
	result := syncResult{total: len(initiativeIDs)}

	if len(initiativeIDs) == 0 {
		s.logger.InfoContext(ctx, "no active initiatives found — nothing to sync")
		return result, nil
	}

	// Step 2 — fetch all Ledger balances in one HTTP call
	rawBalances, err := s.ledger.GetAllBalances(ctx)
	if err != nil {
		return syncResult{}, fmt.Errorf("fetch ledger balances: %w", err)
	}

	// Step 3 — build O(1) lookup map
	balanceIndex := buildBalanceIndex(rawBalances)

	// Step 4 — collect unique org/user IDs across all balances for bulk DB lookup
	orgIDs, userIDs := collectSponsorIDs(rawBalances)

	// Step 5 — bulk lookup org and user details for sponsor enrichment
	orgMap, err := s.repo.GetOrganizationsByIDs(ctx, orgIDs)
	if err != nil {
		return syncResult{}, fmt.Errorf("fetch organization details for sponsors: %w", err)
	}
	userMap, err := s.repo.GetUsersByIDs(ctx, userIDs)
	if err != nil {
		return syncResult{}, fmt.Errorf("fetch user details for sponsors: %w", err)
	}

	// Step 6 — map and enrich each matching initiative
	toUpsert := make([]models.LedgerStats, 0, len(initiativeIDs))
	for _, id := range initiativeIDs {
		raw, ok := balanceIndex[id]
		if !ok {
			result.skipped++
			continue
		}
		toUpsert = append(toUpsert, mapBalance(raw, orgMap, userMap))
		result.matched++
	}

	// Step 7 — bulk upsert
	if len(toUpsert) > 0 {
		n, err := s.repo.BulkUpsertLedgerStats(ctx, toUpsert)
		if err != nil {
			return syncResult{}, fmt.Errorf("bulk upsert initiative_ledger_stats: %w", err)
		}
		result.upserted = n
	}

	return result, nil
}

// buildBalanceIndex returns a map keyed by projectID for O(1) lookups.
// Entries with an empty projectID are silently dropped.
func buildBalanceIndex(balances []models.LedgerRawBalance) map[string]models.LedgerRawBalance {
	idx := make(map[string]models.LedgerRawBalance, len(balances))
	for _, b := range balances {
		if b.ProjectID == "" {
			continue
		}
		idx[b.ProjectID] = b
	}
	return idx
}

// collectSponsorIDs extracts unique, non-empty org and user IDs from the
// sponsors field of every balance entry.  These are used for a single bulk
// SQL lookup to enrich the sponsors JSONB before writing.
func collectSponsorIDs(balances []models.LedgerRawBalance) (orgIDs, userIDs []string) {
	orgSet := make(map[string]struct{})
	userSet := make(map[string]struct{})

	for i := range balances {
		for _, s := range balances[i].Sponsors.Orgs {
			if s.ID != "" {
				orgSet[s.ID] = struct{}{}
			}
		}
		for _, s := range balances[i].Sponsors.Individuals {
			if s.ID != "" {
				userSet[s.ID] = struct{}{}
			}
		}
	}

	orgIDs = make([]string, 0, len(orgSet))
	for id := range orgSet {
		orgIDs = append(orgIDs, id)
	}
	userIDs = make([]string, 0, len(userSet))
	for id := range userSet {
		userIDs = append(userIDs, id)
	}
	return orgIDs, userIDs
}

// mapBalance converts a LedgerRawBalance into a LedgerStats row ready for
// upsert, normalising Ledger fields and enriching sponsors.
//
// Normalisation rules:
//   - TotalDebit and FeeBalance are stored negative by Ledger; ABS both.
//   - TotalRaisedCents = max(raw.TotalCredit, sponsorPositiveSum). Ledger
//     sometimes records fund disbursements as negative-amount "credit"
//     transactions (txnType=credit, amount<0), which corrupts raw.TotalCredit.
//     The sponsor list carries per-entity totals; summing positive entries
//     recovers the actual donated amount.  When all credits are positive,
//     raw.TotalCredit >= sponsorPositiveSum (possible unattributed donations)
//     so we use raw.TotalCredit unchanged.
//   - Negative credits also represent real expenses; their absolute value is
//     added to TotalDebitedCents so "total expenses" reflects them correctly.
func mapBalance(
	raw models.LedgerRawBalance,
	orgMap map[string]models.Organization,
	userMap map[string]models.User,
) models.LedgerStats {
	totalDebit := raw.TotalDebit
	if totalDebit < 0 {
		totalDebit = -totalDebit
	}
	feeBalance := raw.FeeBalance
	if feeBalance < 0 {
		feeBalance = -feeBalance
	}

	// Sum positive sponsor totals to reconstruct the true donated amount.
	var sponsorPositiveSum int64
	for _, o := range raw.Sponsors.Orgs {
		if o.Total > 0 {
			sponsorPositiveSum += o.Total
		}
	}
	for _, u := range raw.Sponsors.Individuals {
		if u.Total > 0 {
			sponsorPositiveSum += u.Total
		}
	}
	totalRaised := raw.TotalCredit
	if sponsorPositiveSum > totalRaised {
		totalRaised = sponsorPositiveSum
	}
	// Any gap between sponsorPositiveSum and raw.TotalCredit represents
	// disbursements recorded as negative credits; treat them as expenses.
	if negCreditAbs := sponsorPositiveSum - raw.TotalCredit; negCreditAbs > 0 {
		totalDebit += negCreditAbs
	}

	// Enrich first so the supporter count is derived from the same data that
	// gets persisted to DB. enrichSponsors already drops entries with empty
	// IDs; the dedup set below guards against any upstream duplicates so the
	// count stays consistent with the sponsors JSONB column.
	// Only entries with a positive total are counted as supporters — negative
	// totals represent expense payouts to recipients, not donor contributions.
	enriched := enrichSponsors(raw.Sponsors, orgMap, userMap)
	seen := make(map[string]struct{}, len(enriched.Orgs)+len(enriched.Individuals))
	for _, o := range enriched.Orgs {
		if o.Total > 0 {
			seen[o.ID] = struct{}{}
		}
	}
	for _, u := range enriched.Individuals {
		if u.Total > 0 {
			seen[u.ID] = struct{}{}
		}
	}

	return models.LedgerStats{
		InitiativeID:          raw.ProjectID,
		TotalRaisedCents:      totalRaised,
		TotalDebitedCents:     totalDebit,
		TotalBalanceCents:     raw.TotalBalance,
		AvailableBalanceCents: raw.AvailableBalance,
		FeeBalanceCents:       feeBalance,
		Supporters:            len(seen),
		Sponsors:              enriched,
	}
}

// enrichSponsors replaces raw org/user ID slices with name + logo/avatar data
// fetched from CF DB.  Entries whose IDs are not found in the DB are included
// with an empty Name so no sponsor data is silently dropped.
func enrichSponsors(
	raw models.LedgerRawSponsors,
	orgMap map[string]models.Organization,
	userMap map[string]models.User,
) models.LedgerSponsorList {
	orgs := make([]models.LedgerSponsorOrg, 0, len(raw.Orgs))
	for _, s := range raw.Orgs {
		if s.ID == "" {
			continue
		}
		entry := models.LedgerSponsorOrg{ID: s.ID, Total: s.Total}
		if org, ok := orgMap[s.ID]; ok {
			entry.Name = org.Name
			entry.AvatarURL = org.AvatarURL
		}
		orgs = append(orgs, entry)
	}

	individuals := make([]models.LedgerSponsorUser, 0, len(raw.Individuals))
	for _, s := range raw.Individuals {
		if s.ID == "" {
			continue
		}
		entry := models.LedgerSponsorUser{ID: s.ID, Total: s.Total}
		if user, ok := userMap[s.ID]; ok {
			entry.Name = user.Name
			entry.AvatarURL = user.AvatarURL
		}
		individuals = append(individuals, entry)
	}

	return models.LedgerSponsorList{Orgs: orgs, Individuals: individuals}
}
