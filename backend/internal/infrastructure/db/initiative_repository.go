// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var initiativeTracer = otel.Tracer("initiatives-db")

// InitiativeRepository implements domain.InitiativeRepository against PostgreSQL.
type InitiativeRepository struct {
	pool *pgxpool.Pool
}

// NewInitiativeRepository creates a new InitiativeRepository.
func NewInitiativeRepository(pool *pgxpool.Pool) *InitiativeRepository {
	return &InitiativeRepository{pool: pool}
}

// GetByID retrieves a single initiative by its UUID primary key.
func (r *InitiativeRepository) GetByID(ctx context.Context, id string) (*models.Initiative, error) {
	ctx, span := initiativeTracer.Start(ctx, "db.initiatives.GetByID")
	defer span.End()
	span.SetAttributes(attribute.String("db.initiative_id", id))

	const q = `
		SELECT id, initiative_type, source_dynamo_table, owner_id,
		       name, slug, status, industry, description, color,
		       logo_url, website_url, coc_url,
		       stripe_plan_id, stripe_product_id,
		       amount_raised_in_cents, accept_funding,
		       cii_project_id, jobspring_project_id, stacks_identifier,
		       eventbrite_url, application_url, event_start_date, event_end_date,
		       country, city, is_online,
		       created_on, updated_on
		FROM initiatives
		WHERE id = $1`

	row := r.pool.QueryRow(ctx, q, id)
	return scanInitiative(row)
}

// GetBySlug retrieves a single initiative by its URL slug.
func (r *InitiativeRepository) GetBySlug(ctx context.Context, slug string) (*models.Initiative, error) {
	ctx, span := initiativeTracer.Start(ctx, "db.initiatives.GetBySlug")
	defer span.End()
	span.SetAttributes(attribute.String("db.initiative_slug", slug))

	const q = `
		SELECT id, initiative_type, source_dynamo_table, owner_id,
		       name, slug, status, industry, description, color,
		       logo_url, website_url, coc_url,
		       stripe_plan_id, stripe_product_id,
		       amount_raised_in_cents, accept_funding,
		       cii_project_id, jobspring_project_id, stacks_identifier,
		       eventbrite_url, application_url, event_start_date, event_end_date,
		       country, city, is_online,
		       created_on, updated_on
		FROM initiatives
		WHERE slug = $1`

	row := r.pool.QueryRow(ctx, q, slug)
	return scanInitiative(row)
}

// List retrieves initiatives matching the filter with pagination.
func (r *InitiativeRepository) List(ctx context.Context, filter models.InitiativeFilter) ([]models.Initiative, *models.PaginationMeta, error) {
	ctx, span := initiativeTracer.Start(ctx, "db.initiatives.List")
	defer span.End()

	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	args := []any{}
	argN := 1
	where := "WHERE 1=1"

	if filter.OwnerID != "" {
		where += fmt.Sprintf(" AND owner_id = $%d", argN)
		args = append(args, filter.OwnerID)
		argN++
	}
	if filter.InitiativeType != "" {
		where += fmt.Sprintf(" AND initiative_type = $%d", argN)
		args = append(args, filter.InitiativeType)
		argN++
	}
	if filter.Status != "" {
		where += fmt.Sprintf(" AND status = $%d", argN)
		args = append(args, filter.Status)
		argN++
	}
	if filter.Search != "" {
		where += fmt.Sprintf(" AND (name ILIKE $%d OR description ILIKE $%d)", argN, argN)
		args = append(args, "%"+filter.Search+"%")
		argN++
	}

	// Count query
	countQuery := "SELECT COUNT(*) FROM initiatives " + where
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		span.RecordError(err)
		return nil, nil, fmt.Errorf("count initiatives: %w", err)
	}

	// Data query
	args = append(args, limit, offset)
	dataQuery := fmt.Sprintf(`
		SELECT id, initiative_type, source_dynamo_table, owner_id,
		       name, slug, status, industry, description, color,
		       logo_url, website_url, coc_url,
		       stripe_plan_id, stripe_product_id,
		       amount_raised_in_cents, accept_funding,
		       cii_project_id, jobspring_project_id, stacks_identifier,
		       eventbrite_url, application_url, event_start_date, event_end_date,
		       country, city, is_online,
		       created_on, updated_on
		FROM initiatives %s
		ORDER BY created_on DESC
		LIMIT $%d OFFSET $%d`, where, argN, argN+1)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		span.RecordError(err)
		return nil, nil, fmt.Errorf("list initiatives: %w", err)
	}
	defer rows.Close()

	var initiatives []models.Initiative
	for rows.Next() {
		i, err := scanInitiativeRow(rows)
		if err != nil {
			return nil, nil, fmt.Errorf("scan initiative: %w", err)
		}
		initiatives = append(initiatives, *i)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate initiatives: %w", err)
	}

	meta := &models.PaginationMeta{Total: total, Limit: limit, Offset: offset}
	return initiatives, meta, nil
}

// Create inserts a new initiative row.
func (r *InitiativeRepository) Create(ctx context.Context, i *models.Initiative) (*models.Initiative, error) {
	ctx, span := initiativeTracer.Start(ctx, "db.initiatives.Create")
	defer span.End()

	const q = `
		INSERT INTO initiatives
		       (initiative_type, owner_id, name, slug, status, industry,
		        description, color, logo_url, website_url, coc_url,
		        stripe_plan_id, stripe_product_id,
		        amount_raised_in_cents, accept_funding)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		RETURNING id, initiative_type, source_dynamo_table, owner_id,
		          name, slug, status, industry, description, color,
		          logo_url, website_url, coc_url,
		          stripe_plan_id, stripe_product_id,
		          amount_raised_in_cents, accept_funding,
		          cii_project_id, jobspring_project_id, stacks_identifier,
		          eventbrite_url, application_url, event_start_date, event_end_date,
		          country, city, is_online,
		          created_on, updated_on`

	row := r.pool.QueryRow(ctx, q,
		i.InitiativeType, i.OwnerID, i.Name, nullableString(i.Slug), nullableString(i.Status),
		nullableString(i.Industry), nullableString(i.Description), nullableString(i.Color),
		nullableString(i.LogoURL), nullableString(i.WebsiteURL), nullableString(i.CocURL),
		nullableString(i.StripePlanID), nullableString(i.StripeProductID),
		i.AmountRaisedCents, i.AcceptFunding,
	)
	created, err := scanInitiative(row)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("create initiative: %w", err)
	}
	return created, nil
}

// Update applies changes to an existing initiative.
func (r *InitiativeRepository) Update(ctx context.Context, i *models.Initiative) (*models.Initiative, error) {
	ctx, span := initiativeTracer.Start(ctx, "db.initiatives.Update")
	defer span.End()
	span.SetAttributes(attribute.String("db.initiative_id", i.ID))

	const q = `
		UPDATE initiatives SET
		    name           = $2,
		    slug           = $3,
		    status         = $4,
		    industry       = $5,
		    description    = $6,
		    color          = $7,
		    logo_url       = $8,
		    website_url    = $9,
		    coc_url        = $10,
		    accept_funding = $11
		WHERE id = $1
		RETURNING id, initiative_type, source_dynamo_table, owner_id,
		          name, slug, status, industry, description, color,
		          logo_url, website_url, coc_url,
		          stripe_plan_id, stripe_product_id,
		          amount_raised_in_cents, accept_funding,
		          cii_project_id, jobspring_project_id, stacks_identifier,
		          eventbrite_url, application_url, event_start_date, event_end_date,
		          country, city, is_online,
		          created_on, updated_on`

	row := r.pool.QueryRow(ctx, q,
		i.ID, i.Name, nullableString(i.Slug), nullableString(i.Status),
		nullableString(i.Industry), nullableString(i.Description), nullableString(i.Color),
		nullableString(i.LogoURL), nullableString(i.WebsiteURL), nullableString(i.CocURL),
		i.AcceptFunding,
	)
	updated, err := scanInitiative(row)
	if err != nil {
		if errors.Is(err, domain.ErrInitiativeNotFound) {
			return nil, domain.ErrInitiativeNotFound
		}
		span.RecordError(err)
		return nil, fmt.Errorf("update initiative: %w", err)
	}
	return updated, nil
}

// Delete removes an initiative by ID.
func (r *InitiativeRepository) Delete(ctx context.Context, id string) error {
	ctx, span := initiativeTracer.Start(ctx, "db.initiatives.Delete")
	defer span.End()
	span.SetAttributes(attribute.String("db.initiative_id", id))

	tag, err := r.pool.Exec(ctx, "DELETE FROM initiatives WHERE id = $1", id)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("delete initiative: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrInitiativeNotFound
	}
	return nil
}

// ListGoals retrieves all goals for an initiative, ordered by sort_order.
func (r *InitiativeRepository) ListGoals(ctx context.Context, initiativeID string) ([]models.Goal, error) {
	ctx, span := initiativeTracer.Start(ctx, "db.initiatives.ListGoals")
	defer span.End()
	span.SetAttributes(attribute.String("db.initiative_id", initiativeID))

	const q = `
		SELECT id, initiative_id, name, amount_in_cents, allocation,
		       repo_link, description, color, icon, sort_order,
		       created_on, updated_on
		FROM initiative_goals
		WHERE initiative_id = $1
		ORDER BY sort_order ASC`

	rows, err := r.pool.Query(ctx, q, initiativeID)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("list goals: %w", err)
	}
	defer rows.Close()

	var goals []models.Goal
	for rows.Next() {
		var g models.Goal
		if err := rows.Scan(
			&g.ID, &g.InitiativeID, &g.Name, &g.AmountInCents, &g.Allocation,
			&g.RepoLink, &g.Description, &g.Color, &g.Icon, &g.SortOrder,
			&g.CreatedOn, &g.UpdatedOn,
		); err != nil {
			return nil, fmt.Errorf("scan goal: %w", err)
		}
		goals = append(goals, g)
	}
	return goals, rows.Err()
}

// ── helpers ─────────────────────────────────────────────────────────────────

type scanner interface {
	Scan(dest ...any) error
}

func scanInitiative(row scanner) (*models.Initiative, error) {
	i := &models.Initiative{}
	// Use pointer intermediaries for all nullable columns so pgx v5 can scan NULLs.
	var (
		sourceDynamoTable, slug, status, industry, description, color *string
		logoURL, websiteURL, cocURL, stripePlanID, stripeProductID    *string
		ciiProjectID, jobspringProjectID, stacksIdentifier            *string
		eventbriteURL, applicationURL, country, city                  *string
		acceptFunding, isOnline                                       *bool
		createdOn, updatedOn                                          *time.Time
	)
	err := row.Scan(
		&i.ID, &i.InitiativeType, &sourceDynamoTable, &i.OwnerID,
		&i.Name, &slug, &status, &industry, &description, &color,
		&logoURL, &websiteURL, &cocURL,
		&stripePlanID, &stripeProductID,
		&i.AmountRaisedCents, &acceptFunding,
		&ciiProjectID, &jobspringProjectID, &stacksIdentifier,
		&eventbriteURL, &applicationURL, &i.EventStartDate, &i.EventEndDate,
		&country, &city, &isOnline,
		&createdOn, &updatedOn,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrInitiativeNotFound
		}
		return nil, err
	}
	i.SourceDynamoTable = derefString(sourceDynamoTable)
	i.Slug = derefString(slug)
	i.Status = derefString(status)
	i.Industry = derefString(industry)
	i.Description = derefString(description)
	i.Color = derefString(color)
	i.LogoURL = derefString(logoURL)
	i.WebsiteURL = derefString(websiteURL)
	i.CocURL = derefString(cocURL)
	i.StripePlanID = derefString(stripePlanID)
	i.StripeProductID = derefString(stripeProductID)
	i.CiiProjectID = derefString(ciiProjectID)
	i.JobspringProjectID = derefString(jobspringProjectID)
	i.StacksIdentifier = derefString(stacksIdentifier)
	i.EventbriteURL = derefString(eventbriteURL)
	i.ApplicationURL = derefString(applicationURL)
	i.Country = derefString(country)
	i.City = derefString(city)
	if acceptFunding != nil {
		i.AcceptFunding = *acceptFunding
	}
	if isOnline != nil {
		i.IsOnline = *isOnline
	}
	if createdOn != nil {
		i.CreatedOn = *createdOn
	}
	if updatedOn != nil {
		i.UpdatedOn = *updatedOn
	}
	return i, nil
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func scanInitiativeRow(row pgx.Rows) (*models.Initiative, error) {
	return scanInitiative(row)
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
