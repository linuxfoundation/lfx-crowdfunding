// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"log/slog"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

// Donor privacy (#466): a donor can hide their name and email on donor lists.
// The choice lives on CF donations and subscriptions only, so ledger rows are
// matched back to them by Stripe ID (exact, per row) or, where the ledger
// doesn't return those IDs (sponsor totals, platform recent donations), by
// donor and initiative: a donor with any anonymous donation to an initiative
// is hidden on that initiative's aggregates. Lookup failures hide the person
// rather than risk showing them.

// anonymousAvatarURL is the avatar shown for a hidden donor. It is the same for
// everyone so it can't link a donor's anonymous rows to each other.
func anonymousAvatarURL() string {
	return generatedAvatarURL(anonymousName, anonymousName)
}

// ledgerRefIDs collects the Stripe IDs of a page of ledger rows.
func ledgerRefIDs(txns []models.Transaction) (sourceTxnIDs, subscriptionIDs []string) {
	seen := map[string]bool{}
	for _, t := range txns {
		if id := t.LedgerSubscriptionID; id != "" && !seen[id] {
			seen[id] = true
			subscriptionIDs = append(subscriptionIDs, id)
		}
		if id := t.LedgerSourceTxnID; id != "" && !seen[id] {
			seen[id] = true
			sourceTxnIDs = append(sourceTxnIDs, id)
		}
	}
	return sourceTxnIDs, subscriptionIDs
}

// matchLedgerRows sets Anonymous on each row from its CF subscription (recurring
// rows) or donation (one-time rows). With own set, rows also carry the CF ID the
// donor toggles. A nil repo matches nothing.
func matchLedgerRows(ctx context.Context, repo domain.DonorPrivacyRepository, txns []models.Transaction, own bool) error {
	if repo == nil || len(txns) == 0 {
		return nil
	}
	sourceTxnIDs, subscriptionIDs := ledgerRefIDs(txns)
	if len(sourceTxnIDs) == 0 && len(subscriptionIDs) == 0 {
		return nil
	}
	matches, err := repo.MatchLedgerRefs(ctx, sourceTxnIDs, subscriptionIDs)
	if err != nil {
		return err
	}
	for i := range txns {
		t := &txns[i]
		if ref, ok := matches.Subscriptions[t.LedgerSubscriptionID]; ok && t.LedgerSubscriptionID != "" {
			t.Anonymous = ref.Anonymous
			if own {
				t.SubscriptionID = ref.ID
			}
			continue
		}
		if ref, ok := matches.Donations[t.LedgerSourceTxnID]; ok && t.LedgerSourceTxnID != "" {
			t.Anonymous = ref.Anonymous
			if own {
				t.DonationID = ref.ID
			}
		}
	}
	return nil
}

// markPublicLedgerRows flags anonymous rows on a list other people see. If the
// lookup fails every row is flagged, so no hidden donor is shown.
func markPublicLedgerRows(ctx context.Context, repo domain.DonorPrivacyRepository, txns []models.Transaction) {
	if err := matchLedgerRows(ctx, repo, txns, false); err != nil {
		slog.WarnContext(ctx, "donor privacy lookup failed; hiding every donor on this page", "error", err)
		for i := range txns {
			txns[i].Anonymous = true
		}
	}
}

// markOwnLedgerRows flags the caller's own rows and attaches the CF IDs used to
// change the setting. A failed lookup only drops the toggles.
func markOwnLedgerRows(ctx context.Context, repo domain.DonorPrivacyRepository, txns []models.Transaction) {
	if err := matchLedgerRows(ctx, repo, txns, true); err != nil {
		slog.WarnContext(ctx, "donor privacy lookup failed for own donations", "error", err)
	}
}

// anonymousDonorFilter returns a check for "is this ledger user hidden on this
// initiative". A nil repo hides nobody; a failed lookup hides every individual.
func anonymousDonorFilter(ctx context.Context, repo domain.DonorPrivacyRepository, initiativeIDs []string) func(initiativeID, ledgerUserID string) bool {
	if repo == nil || len(initiativeIDs) == 0 {
		return func(string, string) bool { return false }
	}
	hidden, err := repo.AnonymousDonors(ctx, initiativeIDs)
	if err != nil {
		slog.WarnContext(ctx, "anonymous donor lookup failed; hiding every individual donor", "error", err)
		return func(string, string) bool { return true }
	}
	return func(initiativeID, ledgerUserID string) bool {
		return hidden[initiativeID][ledgerUserID]
	}
}
