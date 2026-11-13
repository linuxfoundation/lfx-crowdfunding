// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"log/slog"

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
	LookupB2BOrgByWebsite(ctx context.Context, domain string) (models.B2BOrgMatch, error)
	// GetB2BOrgLogo returns the org's current logo; found is false when
	// member-service no longer knows the org.
	GetB2BOrgLogo(ctx context.Context, sfid string) (logoURL string, found bool, err error)
}

// logoStore holds the logos of matched b2b_orgs (#472); domain.OrgLogoRepository.
type logoStore interface {
	MatchedSFIDs(ctx context.Context) ([]string, error)
	SetOrgLogo(ctx context.Context, sfid, logoURL string) error
}

type subscriptionPatcher interface {
	// PatchLiveSubscription writes metadata to the subscription's current item and
	// the subscription. It reports false, writing nothing, when Stripe no longer
	// bills the subscription.
	PatchLiveSubscription(ctx context.Context, subscriptionID, itemID string, metadata map[string]string) (bool, error)
}

type result struct {
	domains, matched, noMatch, lookupFailed int
	donations, subscriptions                int64
	stripePatched, stripeSkipped            int
	stripeFailed                            int
}

// logoResult counts one logo refresh.
type logoResult struct {
	orgs, stored, cleared, lookupFailed int
}

type backfiller struct {
	store  store
	logos  logoStore
	lookup resolver
	stripe subscriptionPatcher
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
		match, err := b.lookup.LookupB2BOrgByWebsite(ctx, domain)
		switch {
		case err != nil:
			res.lookupFailed++
			b.logger.Warn("b2b_org lookup failed", "org_domain", domain, "error", err)
			continue
		case match.SFID == "":
			res.noMatch++
			continue
		}
		sfid := match.SFID
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

// RefreshLogos re-reads the logo of every matched b2b_org from member-service
// and stores it (#472), so logos changed in Self Serve reach donor lists within
// a day. Run it after Run, which matches new domains. A failed lookup keeps the
// stored logo; an org member-service no longer knows, or one without a logo,
// loses its row and shows initials.
func (b *backfiller) RefreshLogos(ctx context.Context) (logoResult, error) {
	var res logoResult
	sfids, err := b.logos.MatchedSFIDs(ctx)
	if err != nil {
		return res, err
	}
	res.orgs = len(sfids)

	for _, sfid := range sfids {
		logoURL, found, err := b.lookup.GetB2BOrgLogo(ctx, sfid)
		if err != nil {
			res.lookupFailed++
			b.logger.Warn("b2b_org logo lookup failed, keeping stored logo", "b2b_org_sfid", sfid, "error", err)
			continue
		}
		if !found {
			logoURL = ""
		}
		if logoURL == "" {
			res.cleared++
		} else {
			res.stored++
		}
		if b.dryRun {
			continue
		}
		if err := b.logos.SetOrgLogo(ctx, sfid, logoURL); err != nil {
			return res, err
		}
	}
	return res, nil
}

// patchStripe writes b2b_org_sfid to a subscription's Stripe metadata so
// ledger-service's hook carries it on renewals. It reports whether the row should
// get the SFID: false for a Stripe failure, which the next run retries.
func (b *backfiller) patchStripe(ctx context.Context, sub unmatchedSub, sfid string, res *result) bool {
	// A canceled subscription never charges again, and Stripe rejects updates to it.
	if sub.Status == models.SubscriptionStatusCanceled || sub.StripeSubscriptionID == "" {
		res.stripeSkipped++
		return true
	}
	live, err := b.stripe.PatchLiveSubscription(ctx, sub.StripeSubscriptionID, sub.StripeSubscriptionItemID, map[string]string{"b2b_org_sfid": sfid})
	switch {
	case err != nil:
		// Any Stripe error, including a 400 or 404, is retried: canceled
		// subscriptions stay retrievable, so a 404 points at the wrong key or account.
		res.stripeFailed++
		b.logger.Warn("stripe metadata update failed, will retry", "subscription_id", sub.ID, "stripe_subscription_id", sub.StripeSubscriptionID, "error", err)
		return false
	case !live:
		// CF's status can be stale (#438): Stripe has already ended it for good.
		res.stripeSkipped++
		b.logger.Info("stripe subscription not live, filling row only", "subscription_id", sub.ID, "stripe_subscription_id", sub.StripeSubscriptionID)
		return true
	}
	res.stripePatched++
	return true
}
