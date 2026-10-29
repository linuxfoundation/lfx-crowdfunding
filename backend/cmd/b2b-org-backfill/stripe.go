// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/stripe/stripe-go/v85"
)

// stripePatcher checks a subscription's status in Stripe before writing to it,
// since CF's status and stored item ID can be stale (#438).
type stripePatcher struct {
	api *stripe.Client
}

func (p stripePatcher) PatchLiveSubscription(ctx context.Context, subscriptionID, itemID string, metadata map[string]string) (bool, error) {
	sub, err := p.api.V1Subscriptions.Retrieve(ctx, subscriptionID, nil)
	if err != nil {
		return false, fmt.Errorf("stripe get subscription: %w", err)
	}
	if sub.Status != stripe.SubscriptionStatusActive && sub.Status != stripe.SubscriptionStatusPastDue {
		return false, nil
	}
	item := currentItem(sub, itemID)
	if item == "" {
		return false, errors.New("stripe subscription has no items")
	}
	// Ledger's charge.succeeded hook reads the invoice line, which carries the item metadata.
	if _, err := p.api.V1SubscriptionItems.Update(ctx, item, &stripe.SubscriptionItemUpdateParams{Metadata: metadata}); err != nil {
		return false, fmt.Errorf("stripe update subscription item metadata: %w", err)
	}
	if _, err := p.api.V1Subscriptions.Update(ctx, subscriptionID, &stripe.SubscriptionUpdateParams{Metadata: metadata}); err != nil {
		return false, fmt.Errorf("stripe update subscription metadata: %w", err)
	}
	return true, nil
}

// currentItem returns itemID if the subscription still has it, else its first
// item, or "" when it has none.
func currentItem(sub *stripe.Subscription, itemID string) string {
	if sub.Items == nil || len(sub.Items.Data) == 0 {
		return ""
	}
	for _, it := range sub.Items.Data {
		if it.ID == itemID {
			return it.ID
		}
	}
	return sub.Items.Data[0].ID
}
