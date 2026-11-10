// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var donorPrivacyTracer = otel.Tracer("donor-privacy-db")

// DonorPrivacyRepository implements domain.DonorPrivacyRepository against PostgreSQL.
type DonorPrivacyRepository struct {
	pool *pgxpool.Pool
}

// NewDonorPrivacyRepository creates a new DonorPrivacyRepository.
func NewDonorPrivacyRepository(pool *pgxpool.Pool) *DonorPrivacyRepository {
	return &DonorPrivacyRepository{pool: pool}
}

// MatchLedgerRefs maps ledger rows' Stripe IDs to CF donations and subscriptions.
// A one-time ledger row carries the charge ID, or the PaymentIntent ID when the
// charge wasn't expanded yet, so donations match on either column.
func (r *DonorPrivacyRepository) MatchLedgerRefs(ctx context.Context, sourceTxnIDs, stripeSubscriptionIDs []string) (*models.LedgerRefMatches, error) {
	ctx, span := donorPrivacyTracer.Start(ctx, "db.donor_privacy.MatchLedgerRefs")
	defer span.End()

	out := &models.LedgerRefMatches{
		Donations:     map[string]models.LedgerRef{},
		Subscriptions: map[string]models.LedgerRef{},
	}
	if len(sourceTxnIDs) > 0 {
		rows, err := r.pool.Query(ctx, `
			SELECT CAST(id AS text), stripe_charge_id, stripe_payment_intent_id, anonymous
			FROM donations
			WHERE stripe_charge_id = ANY($1) OR stripe_payment_intent_id = ANY($1)`, sourceTxnIDs)
		if err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("match ledger donations: %w", err)
		}
		for rows.Next() {
			var id string
			var chargeID, piID *string
			var anonymous bool
			if err := rows.Scan(&id, &chargeID, &piID, &anonymous); err != nil {
				rows.Close()
				return nil, fmt.Errorf("scan ledger donation match: %w", err)
			}
			ref := models.LedgerRef{ID: id, Anonymous: anonymous}
			if c := derefString(chargeID); c != "" {
				out.Donations[c] = ref
			}
			if p := derefString(piID); p != "" {
				out.Donations[p] = ref
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("iterate ledger donation matches: %w", err)
		}
	}
	if len(stripeSubscriptionIDs) > 0 {
		rows, err := r.pool.Query(ctx, `
			SELECT CAST(id AS text), stripe_subscription_id, anonymous
			FROM subscriptions
			WHERE stripe_subscription_id = ANY($1)`, stripeSubscriptionIDs)
		if err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("match ledger subscriptions: %w", err)
		}
		for rows.Next() {
			var id, stripeID string
			var anonymous bool
			if err := rows.Scan(&id, &stripeID, &anonymous); err != nil {
				rows.Close()
				return nil, fmt.Errorf("scan ledger subscription match: %w", err)
			}
			out.Subscriptions[stripeID] = models.LedgerRef{ID: id, Anonymous: anonymous}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("iterate ledger subscription matches: %w", err)
		}
	}
	return out, nil
}

// AnonymousDonors returns, per initiative, the legacy user IDs (the ledger's
// user key) of donors with any anonymous donation or subscription to it.
func (r *DonorPrivacyRepository) AnonymousDonors(ctx context.Context, initiativeIDs []string) (map[string]map[string]bool, error) {
	ctx, span := donorPrivacyTracer.Start(ctx, "db.donor_privacy.AnonymousDonors")
	defer span.End()

	out := map[string]map[string]bool{}
	if len(initiativeIDs) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT CAST(t.initiative_id AS text), u.legacy_user_id
		FROM (
			SELECT initiative_id, user_id FROM donations
			WHERE anonymous AND CAST(initiative_id AS text) = ANY($1)
			UNION
			SELECT initiative_id, user_id FROM subscriptions
			WHERE anonymous AND CAST(initiative_id AS text) = ANY($1)
		) t
		JOIN users u ON u.id = t.user_id
		WHERE COALESCE(u.legacy_user_id, '') <> ''`, initiativeIDs)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("list anonymous donors: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var initiativeID, legacyUserID string
		if err := rows.Scan(&initiativeID, &legacyUserID); err != nil {
			return nil, fmt.Errorf("scan anonymous donor: %w", err)
		}
		if out[initiativeID] == nil {
			out[initiativeID] = map[string]bool{}
		}
		out[initiativeID][legacyUserID] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate anonymous donors: %w", err)
	}
	return out, nil
}

// AnonymousLedgerUsers returns which of the given legacy user IDs have any
// anonymous donation or subscription, on any initiative.
func (r *DonorPrivacyRepository) AnonymousLedgerUsers(ctx context.Context, ledgerUserIDs []string) (map[string]bool, error) {
	ctx, span := donorPrivacyTracer.Start(ctx, "db.donor_privacy.AnonymousLedgerUsers")
	defer span.End()

	out := map[string]bool{}
	if len(ledgerUserIDs) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT u.legacy_user_id
		FROM users u
		WHERE u.legacy_user_id = ANY($1)
		  AND (EXISTS (SELECT 1 FROM donations d WHERE d.user_id = u.id AND d.anonymous)
		       OR EXISTS (SELECT 1 FROM subscriptions s WHERE s.user_id = u.id AND s.anonymous))`, ledgerUserIDs)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("list anonymous ledger users: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan anonymous ledger user: %w", err)
		}
		out[id] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate anonymous ledger users: %w", err)
	}
	return out, nil
}

// IsPaymentIntentAnonymous reports whether the donation for a PaymentIntent is
// anonymous; false when no donation matches.
func (r *DonorPrivacyRepository) IsPaymentIntentAnonymous(ctx context.Context, paymentIntentID string) (bool, error) {
	ctx, span := donorPrivacyTracer.Start(ctx, "db.donor_privacy.IsPaymentIntentAnonymous")
	defer span.End()

	var anonymous bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM donations WHERE stripe_payment_intent_id = $1 AND anonymous)`,
		paymentIntentID).Scan(&anonymous)
	if err != nil {
		span.RecordError(err)
		return false, fmt.Errorf("check donation anonymity: %w", err)
	}
	return anonymous, nil
}

// IsStripeSubscriptionAnonymous reports whether the subscription for a Stripe
// subscription ID is anonymous; false when none matches.
func (r *DonorPrivacyRepository) IsStripeSubscriptionAnonymous(ctx context.Context, stripeSubscriptionID string) (bool, error) {
	ctx, span := donorPrivacyTracer.Start(ctx, "db.donor_privacy.IsStripeSubscriptionAnonymous")
	defer span.End()

	var anonymous bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM subscriptions WHERE stripe_subscription_id = $1 AND anonymous)`,
		stripeSubscriptionID).Scan(&anonymous)
	if err != nil {
		span.RecordError(err)
		return false, fmt.Errorf("check subscription anonymity: %w", err)
	}
	return anonymous, nil
}

// SetDonationAnonymous updates the caller's own donation.
func (r *DonorPrivacyRepository) SetDonationAnonymous(ctx context.Context, id, userID string, anonymous bool) error {
	ctx, span := donorPrivacyTracer.Start(ctx, "db.donor_privacy.SetDonationAnonymous")
	defer span.End()
	span.SetAttributes(attribute.String("db.donation_id", id))

	err := r.pool.QueryRow(ctx, `
		UPDATE donations SET anonymous = $3, updated_on = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING id`, id, userID, anonymous).Scan(new(any))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrDonationNotFound
	}
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("set donation anonymous: %w", err)
	}
	return nil
}

// SetSubscriptionAnonymous updates the caller's own subscription.
func (r *DonorPrivacyRepository) SetSubscriptionAnonymous(ctx context.Context, id, userID string, anonymous bool) error {
	ctx, span := donorPrivacyTracer.Start(ctx, "db.donor_privacy.SetSubscriptionAnonymous")
	defer span.End()
	span.SetAttributes(attribute.String("db.subscription_id", id))

	err := r.pool.QueryRow(ctx, `
		UPDATE subscriptions SET anonymous = $3, updated_on = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING id`, id, userID, anonymous).Scan(new(any))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrSubscriptionNotFound
	}
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("set subscription anonymous: %w", err)
	}
	return nil
}
