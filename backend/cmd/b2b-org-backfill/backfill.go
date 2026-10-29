// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/stripe/stripe-go/v85"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

// unmatchedSub is an org subscription with no b2b_org SFID yet.
type unmatchedSub struct {
	ID                       string
	Status                   string
	StripeSubscriptionID     string
	StripeSubscriptionItemID string
}

// store is the DB side of the backfill.
type store interface {
	// UnmatchedDomains returns each distinct org_domain with a donation or
	// subscription that has no b2b_org SFID.
	UnmatchedDomains(ctx context.Context) ([]string, error)
	// SetDonationsSFID fills the SFID on the domain's donations that have none.
	SetDonationsSFID(ctx context.Context, domain, sfid string) (int64, error)
	UnmatchedSubscriptions(ctx context.Context, domain string) ([]unmatchedSub, error)
	SetSubscriptionSFID(ctx context.Context, id, sfid string) error
}

type resolver interface {
	LookupB2BOrgByWebsite(ctx context.Context, domain string) (string, error)
}

type subscriptionMetadataUpdater interface {
	UpdateSubscriptionMetadata(ctx context.Context, subscriptionID, itemID string, metadata map[string]string) error
}

type result struct {
	domains, matched, noMatch, lookupFailed int
	donations, subscriptions                int64
	stripePatched, stripeSkipped            int
	stripeFailed                            int
}

type backfiller struct {
	store  store
	lookup resolver
	stripe subscriptionMetadataUpdater
	dryRun bool
	logger *slog.Logger
}

// Run looks up each unmatched domain once and fills its rows. A failed lookup or
// Stripe update leaves the rows empty, so the next run retries them.
func (b *backfiller) Run(ctx context.Context) (result, error) {
	var res result
	domains, err := b.store.UnmatchedDomains(ctx)
	if err != nil {
		return res, err
	}
	res.domains = len(domains)

	for _, domain := range domains {
		sfid, err := b.lookup.LookupB2BOrgByWebsite(ctx, domain)
		switch {
		case err != nil:
			res.lookupFailed++
			b.logger.Warn("b2b_org lookup failed", "org_domain", domain, "error", err)
			continue
		case sfid == "":
			res.noMatch++
			continue
		}
		res.matched++
		b.logger.Info("b2b_org matched", "org_domain", domain, "b2b_org_sfid", sfid, "dry_run", b.dryRun)
		if b.dryRun {
			continue
		}

		n, err := b.store.SetDonationsSFID(ctx, domain, sfid)
		if err != nil {
			return res, err
		}
		res.donations += n

		subs, err := b.store.UnmatchedSubscriptions(ctx, domain)
		if err != nil {
			return res, err
		}
		for _, sub := range subs {
			if !b.patchStripe(ctx, sub, sfid, &res) {
				continue
			}
			if err := b.store.SetSubscriptionSFID(ctx, sub.ID, sfid); err != nil {
				return res, err
			}
			res.subscriptions++
		}
	}
	return res, nil
}

// patchStripe writes b2b_org_sfid to a subscription's Stripe metadata so
// ledger-service's hook carries it on renewals. It reports whether the row should
// get the SFID: false only for a temporary Stripe failure worth retrying.
func (b *backfiller) patchStripe(ctx context.Context, sub unmatchedSub, sfid string, res *result) bool {
	// A canceled subscription never charges again, and Stripe rejects updates to it.
	if sub.Status == models.SubscriptionStatusCanceled || sub.StripeSubscriptionID == "" {
		res.stripeSkipped++
		return true
	}
	err := b.stripe.UpdateSubscriptionMetadata(ctx, sub.StripeSubscriptionID, sub.StripeSubscriptionItemID, map[string]string{"b2b_org_sfid": sfid})
	if err == nil {
		res.stripePatched++
		return true
	}
	// CF's status can be stale (#438): Stripe answers 400/404 for a subscription it
	// has already canceled. Treat that as not live and still fill the row.
	var se *stripe.Error
	if errors.As(err, &se) && (se.HTTPStatusCode == http.StatusBadRequest || se.HTTPStatusCode == http.StatusNotFound) {
		res.stripeSkipped++
		b.logger.Info("stripe subscription not updatable, filling row only", "subscription_id", sub.ID, "stripe_subscription_id", sub.StripeSubscriptionID, "error", err)
		return true
	}
	res.stripeFailed++
	b.logger.Warn("stripe metadata update failed, will retry", "subscription_id", sub.ID, "stripe_subscription_id", sub.StripeSubscriptionID, "error", err)
	return false
}
