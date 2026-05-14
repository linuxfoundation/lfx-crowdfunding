// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package domain defines shared domain errors and repository contracts.
package domain

import (
	"context"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

// InitiativeRepository defines persistence operations for initiatives.
type InitiativeRepository interface {
	GetByID(ctx context.Context, id string) (*models.Initiative, error)
	GetBySlug(ctx context.Context, slug string) (*models.Initiative, error)
	List(ctx context.Context, filter models.InitiativeFilter) ([]models.Initiative, *models.PaginationMeta, error)
	Create(ctx context.Context, initiative *models.Initiative) (*models.Initiative, error)
	Update(ctx context.Context, initiative *models.Initiative) (*models.Initiative, error)
	Delete(ctx context.Context, id string) error
	ListGoals(ctx context.Context, initiativeID string) ([]models.Goal, error)
}

// DonationRepository defines persistence operations for donations.
type DonationRepository interface {
	GetByID(ctx context.Context, id string) (*models.Donation, error)
	ListByInitiative(ctx context.Context, initiativeID string, filter models.DonationFilter) ([]models.Donation, *models.PaginationMeta, error)
	ListByUser(ctx context.Context, userID string, filter models.DonationFilter) ([]models.Donation, *models.PaginationMeta, error)
	Create(ctx context.Context, donation *models.Donation) (*models.Donation, error)
}

// SubscriptionRepository defines persistence operations for subscriptions.
type SubscriptionRepository interface {
	GetByID(ctx context.Context, id string) (*models.Subscription, error)
	ListByInitiative(ctx context.Context, initiativeID string, filter models.SubscriptionFilter) ([]models.Subscription, *models.PaginationMeta, error)
	ListByUser(ctx context.Context, userID string, filter models.SubscriptionFilter) ([]models.Subscription, *models.PaginationMeta, error)
	Create(ctx context.Context, sub *models.Subscription) (*models.Subscription, error)
	Update(ctx context.Context, sub *models.Subscription) (*models.Subscription, error)
}

// OrganizationRepository defines persistence operations for organizations.
type OrganizationRepository interface {
	GetByID(ctx context.Context, id string) (*models.Organization, error)
	ListByOwner(ctx context.Context, ownerID string) ([]models.Organization, error)
}

// UserRepository defines persistence operations for users.
type UserRepository interface {
	GetByUserID(ctx context.Context, userID string) (*models.User, error)
	Upsert(ctx context.Context, user *models.User) (*models.User, error)
}
