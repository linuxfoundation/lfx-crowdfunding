// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package db provides PostgreSQL connection helpers and repositories.
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

// initiativeSelect is the common SELECT + JOIN used by both GetByID and List.
// It joins initiative_ledger_stats for financial data and uses a subquery for
// goals_total_cents to avoid row multiplication from a direct goals JOIN.
const initiativeSelect = `
	SELECT
		i.id, i.initiative_type, i.source_dynamo_table, i.owner_id,
		i.name, i.slug, i.status, i.industry, i.description, i.color,
		i.logo_url, i.website_url, i.coc_url,
		i.stripe_plan_id, i.stripe_product_id,
		i.accept_funding,
		i.cii_project_id, i.jobspring_project_id, i.stacks_identifier,
		i.eventbrite_url, i.application_url, i.event_start_date, i.event_end_date,
		i.country, i.city, i.is_online,
		i.created_on, i.updated_on,
		COALESCE(ls.total_raised_cents, 0)      AS total_raised_cents,
		COALESCE(ls.total_debited_cents, 0)     AS total_disbursed_cents,
		COALESCE(ls.available_balance_cents, 0) AS available_balance_cents,
		COALESCE(ls.supporters, 0)              AS supporters,
		COALESCE((
			SELECT SUM(amount_in_cents)::bigint
			FROM initiative_goals
			WHERE initiative_id = i.id
		), 0)::bigint AS goals_total_cents
	FROM initiatives i
	LEFT JOIN initiative_ledger_stats ls ON ls.initiative_id = i.id`

// GetByID retrieves a single initiative by its UUID primary key.
func (r *InitiativeRepository) GetByID(ctx context.Context, id string) (*models.Initiative, error) {
	ctx, span := initiativeTracer.Start(ctx, "db.initiatives.GetByID")
	defer span.End()
	span.SetAttributes(attribute.String("db.initiative_id", id))

	q := initiativeSelect + " WHERE i.id = $1"
	row := r.pool.QueryRow(ctx, q, id)
	initiative, err := scanInitiative(row)
	if err != nil {
		return nil, err
	}

	goals, err := r.listGoals(ctx, id)
	if err != nil {
		return nil, err
	}
	initiative.Goals = goals
	return initiative, nil
}

// GetBySlug retrieves a single initiative by its URL slug.
func (r *InitiativeRepository) GetBySlug(ctx context.Context, slug string) (*models.Initiative, error) {
	ctx, span := initiativeTracer.Start(ctx, "db.initiatives.GetBySlug")
	defer span.End()
	span.SetAttributes(attribute.String("db.initiative_slug", slug))

	q := initiativeSelect + " WHERE i.slug = $1"
	row := r.pool.QueryRow(ctx, q, slug)
	initiative, err := scanInitiative(row)
	if err != nil {
		return nil, err
	}

	goals, err := r.listGoals(ctx, initiative.ID)
	if err != nil {
		return nil, err
	}
	initiative.Goals = goals
	return initiative, nil
}

// List retrieves initiatives matching the filter with pagination.
func (r *InitiativeRepository) List(ctx context.Context, filter models.InitiativeFilter) ([]*models.Initiative, *models.PaginationMeta, error) {
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
		where += fmt.Sprintf(" AND i.owner_id = $%d", argN)
		args = append(args, filter.OwnerID)
		argN++
	}
	if filter.InitiativeType != "" {
		where += fmt.Sprintf(" AND i.initiative_type = $%d", argN)
		args = append(args, filter.InitiativeType)
		argN++
	}
	if filter.Status != "" {
		where += fmt.Sprintf(" AND i.status = $%d", argN)
		args = append(args, filter.Status)
		argN++
	}
	if filter.Search != "" {
		where += fmt.Sprintf(" AND (i.name ILIKE $%d OR i.description ILIKE $%d)", argN, argN)
		args = append(args, "%"+filter.Search+"%")
		argN++
	}

	// Count — no JOIN needed
	countQuery := "SELECT COUNT(*) FROM initiatives i " + where
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		span.RecordError(err)
		return nil, nil, fmt.Errorf("count initiatives: %w", err)
	}

	// Sort — allowlist prevents SQL injection
	orderCol := "i.created_on"
	switch filter.SortBy {
	case "supporters":
		orderCol = "COALESCE(ls.supporters, 0)"
	case "total_raised":
		orderCol = "COALESCE(ls.total_raised_cents, 0)"
	}
	orderDir := "DESC"
	if filter.SortDir == "asc" {
		orderDir = "ASC"
	}

	args = append(args, limit, offset)
	// When sorting by a financial metric, append created_on+id as tiebreakers for
	// deterministic pagination. When using the default created_on sort, i.id alone
	// is sufficient to break ties (avoids repeating the same column).
	secondarySort := ", i.created_on DESC, i.id"
	if filter.SortBy == "" {
		secondarySort = ", i.id"
	}
	dataQuery := fmt.Sprintf("%s %s ORDER BY %s %s%s LIMIT $%d OFFSET $%d",
		initiativeSelect, where, orderCol, orderDir, secondarySort, argN, argN+1)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		span.RecordError(err)
		return nil, nil, fmt.Errorf("list initiatives: %w", err)
	}
	defer rows.Close()

	var initiatives []*models.Initiative
	var ids []string
	for rows.Next() {
		i, err := scanInitiative(rows)
		if err != nil {
			return nil, nil, fmt.Errorf("scan initiative: %w", err)
		}
		initiatives = append(initiatives, i)
		ids = append(ids, i.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate initiatives: %w", err)
	}

	// Fetch all goals for the returned initiatives in one query
	if len(ids) > 0 {
		goalsByID, err := r.listGoalsForIDs(ctx, ids)
		if err != nil {
			return nil, nil, err
		}
		for _, i := range initiatives {
			if goals, ok := goalsByID[i.ID]; ok {
				i.Goals = goals
			}
		}
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
		        stripe_plan_id, stripe_product_id, accept_funding)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		RETURNING id`

	var id string
	err := r.pool.QueryRow(ctx, q,
		i.InitiativeType, i.OwnerID, i.Name, nullableString(i.Slug), nullableString(i.Status),
		nullableString(i.Industry), nullableString(i.Description), nullableString(i.Color),
		nullableString(i.LogoURL), nullableString(i.WebsiteURL), nullableString(i.CocURL),
		nullableString(i.StripePlanID), nullableString(i.StripeProductID),
		i.AcceptFunding,
	).Scan(&id)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("create initiative: %w", err)
	}
	return r.GetByID(ctx, id)
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
		WHERE id = $1`

	tag, err := r.pool.Exec(ctx, q,
		i.ID, i.Name, nullableString(i.Slug), nullableString(i.Status),
		nullableString(i.Industry), nullableString(i.Description), nullableString(i.Color),
		nullableString(i.LogoURL), nullableString(i.WebsiteURL), nullableString(i.CocURL),
		i.AcceptFunding,
	)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("update initiative: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, domain.ErrInitiativeNotFound
	}
	return r.GetByID(ctx, i.ID)
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

// ── private helpers ──────────────────────────────────────────────────────────

func (r *InitiativeRepository) listGoals(ctx context.Context, initiativeID string) ([]models.Goal, error) {
	const q = `
		SELECT id, initiative_id, name, amount_in_cents, allocation,
		       repo_link, description, color, icon, sort_order,
		       created_on, updated_on
		FROM initiative_goals
		WHERE initiative_id = $1
		ORDER BY sort_order ASC`

	rows, err := r.pool.Query(ctx, q, initiativeID)
	if err != nil {
		return nil, fmt.Errorf("list goals: %w", err)
	}
	defer rows.Close()
	return scanGoals(rows)
}

// listGoalsForIDs fetches goals for a set of initiative IDs in one query,
// returning a map of initiativeID → []Goal.
func (r *InitiativeRepository) listGoalsForIDs(ctx context.Context, ids []string) (map[string][]models.Goal, error) {
	// Build $1,$2,... placeholders
	args := make([]any, len(ids))
	placeholders := ""
	for i, id := range ids {
		if i > 0 {
			placeholders += ","
		}
		placeholders += fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	q := fmt.Sprintf(`
		SELECT id, initiative_id, name, amount_in_cents, allocation,
		       repo_link, description, color, icon, sort_order,
		       created_on, updated_on
		FROM initiative_goals
		WHERE initiative_id IN (%s)
		ORDER BY initiative_id, sort_order ASC`, placeholders)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list goals for ids: %w", err)
	}
	defer rows.Close()

	result := make(map[string][]models.Goal)
	for rows.Next() {
		var g models.Goal
		if err := rows.Scan(
			&g.ID, &g.InitiativeID, &g.Name, &g.AmountInCents, &g.Allocation,
			&g.RepoLink, &g.Description, &g.Color, &g.Icon, &g.SortOrder,
			&g.CreatedOn, &g.UpdatedOn,
		); err != nil {
			return nil, fmt.Errorf("scan goal: %w", err)
		}
		result[g.InitiativeID] = append(result[g.InitiativeID], g)
	}
	return result, rows.Err()
}

func scanGoals(rows pgx.Rows) ([]models.Goal, error) {
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

type scanner interface {
	Scan(dest ...any) error
}

func scanInitiative(row scanner) (*models.Initiative, error) {
	i := &models.Initiative{Goals: []models.Goal{}}
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
		&acceptFunding,
		&ciiProjectID, &jobspringProjectID, &stacksIdentifier,
		&eventbriteURL, &applicationURL, &i.EventStartDate, &i.EventEndDate,
		&country, &city, &isOnline,
		&createdOn, &updatedOn,
		&i.Financials.TotalRaisedCents,
		&i.Financials.TotalDisbursedCents,
		&i.Financials.AvailableBalanceCents,
		&i.Financials.Supporters,
		&i.Financials.GoalsTotalCents,
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

	if i.Financials.GoalsTotalCents > 0 {
		i.Financials.FundedPercent = int(i.Financials.TotalRaisedCents * 100 / i.Financials.GoalsTotalCents)
	}

	return i, nil
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
