// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package db provides PostgreSQL connection helpers and repositories.
package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
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
		), 0)::bigint AS goals_total_cents,
		COALESCE(ls.sponsors, '{}')::jsonb      AS sponsors
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
		if !errors.Is(err, domain.ErrInitiativeNotFound) {
			span.RecordError(err)
		}
		return nil, err
	}

	goals, err := r.listGoals(ctx, id)
	if err != nil {
		span.RecordError(err)
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
		if !errors.Is(err, domain.ErrInitiativeNotFound) {
			span.RecordError(err)
		}
		return nil, err
	}

	goals, err := r.listGoals(ctx, initiative.ID)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	initiative.Goals = goals
	return initiative, nil
}

// GetIDBySlug returns the UUID of the initiative with the given slug.
// Cheaper than GetBySlug — no goals query, no Ledger enrichment.
func (r *InitiativeRepository) GetIDBySlug(ctx context.Context, slug string) (string, error) {
	ctx, span := initiativeTracer.Start(ctx, "db.initiatives.GetIDBySlug")
	defer span.End()
	span.SetAttributes(attribute.String("db.initiative_slug", slug))

	var id string
	err := r.pool.QueryRow(ctx, `SELECT id FROM initiatives WHERE slug = $1 AND LOWER(status) = $2`, slug, models.StatusPublished).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", domain.ErrInitiativeNotFound
		}
		span.RecordError(err)
		return "", fmt.Errorf("get id by slug: %w", err)
	}
	return id, nil
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
	if filter.SortBy == "" || filter.SortBy == "created_on" {
		secondarySort = ", i.id"
	}
	dataQuery := fmt.Sprintf("%s %s ORDER BY %s %s%s LIMIT $%d OFFSET $%d",
		initiativeSelect, where, orderCol, orderDir, secondarySort, argN, argN+1)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		span.RecordError(err)
		return nil, nil, fmt.Errorf("list initiatives: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var initiatives []*models.Initiative
	var ids []string
	for rows.Next() {
		i, err := scanInitiative(rows)
		if err != nil {
			span.RecordError(err)
			return nil, nil, fmt.Errorf("scan initiative: %w", err)
		}
		initiatives = append(initiatives, i)
		ids = append(ids, i.ID)
	}
	if err := rows.Err(); err != nil {
		span.RecordError(err)
		return nil, nil, fmt.Errorf("iterate initiatives: %w", err)
	}

	// Fetch all goals for the returned initiatives in one query
	if len(ids) > 0 {
		goalsByID, err := r.listGoalsForIDs(ctx, ids)
		if err != nil {
			span.RecordError(err)
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

// SQL constants for Create — grouped here so they can be read together and
// linted independently of the surrounding logic.
const (
	insertInitiative = `
		INSERT INTO initiatives
		       (id, initiative_type, owner_id, name, slug, status, industry,
		        description, color, logo_url, website_url, coc_url,
		        stripe_plan_id, stripe_product_id, accept_funding,
		        eventbrite_url, application_url, event_start_date, event_end_date,
		        country, city, is_online)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,
		        $16,$17,$18,$19,$20,$21,$22)`

	insertGoal = `
		INSERT INTO initiative_goals
		       (id, initiative_id, name, amount_in_cents, allocation,
		        repo_link, description, color, icon, sort_order)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9)`

	insertBeneficiary = `
		INSERT INTO initiative_beneficiaries (id, initiative_id, name, email)
		VALUES (gen_random_uuid(), $1, $2, $3)`

	insertCustomWebsite = `
		INSERT INTO initiative_custom_websites (id, initiative_id, name, url)
		VALUES (gen_random_uuid(), $1, $2, $3)`

	insertContributor = `
		INSERT INTO initiative_contributors (id, initiative_id, name, email)
		VALUES (gen_random_uuid(), $1, $2, $3)`

	insertMentor = `
		INSERT INTO initiative_mentors
		       (id, initiative_id, name, email, avatar_url, introduction)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5)`

	insertProgramTerm = `
		INSERT INTO initiative_program_info_terms
		       (id, initiative_id, term, sort_order)
		VALUES (gen_random_uuid(), $1, $2, $3)`

	insertProgramSkill = `
		INSERT INTO initiative_program_info_skills (id, initiative_id, skill)
		VALUES (gen_random_uuid(), $1, $2)
		ON CONFLICT (initiative_id, skill) DO NOTHING`

	insertProgramConfig = `
		INSERT INTO initiative_program_info_config (initiative_id, terms_conditions)
		VALUES ($1, $2)`

	insertProgramCustomTerm = `
		INSERT INTO initiative_program_info_custom_term
		       (initiative_id, term_name, start_month, end_month, year)
		VALUES ($1, $2, $3, $4, $5)`

	insertSponsorshipTier = `
		INSERT INTO initiative_sponsorship_tiers
		       (id, initiative_id, name, description, color, icon, minimum, sort_order)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7)`

	insertOSTIFDetail = `
		INSERT INTO initiative_ostif_detail
		       (initiative_id, monetization_strategy, current_security_strategy,
		        license_type, total_budget_in_cents, terms_conditions)
		VALUES ($1, $2, $3, $4, $5, $6)`

	insertContact = `
		INSERT INTO initiative_contacts
		       (id, initiative_id, contact_type, first_name, last_name, email,
		        phone_number, other_contact_option, preferred_contact_method)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8)`

	insertGitHubStats = `
		INSERT INTO initiative_github_stats (initiative_id, forks, stars, open_issues)
		VALUES ($1, 0, 0, 0)`

	insertEntityDetails = `
		INSERT INTO initiative_entity_details (initiative_id, details)
		VALUES ($1, $2)`

	// DELETE constants used by the transactional Update to replace child rows.
	deleteGoals             = `DELETE FROM initiative_goals WHERE initiative_id = $1`
	deleteBeneficiaries     = `DELETE FROM initiative_beneficiaries WHERE initiative_id = $1`
	deleteCustomWebsites    = `DELETE FROM initiative_custom_websites WHERE initiative_id = $1`
	deleteContributors      = `DELETE FROM initiative_contributors WHERE initiative_id = $1`
	deleteMentors           = `DELETE FROM initiative_mentors WHERE initiative_id = $1`
	deleteProgramTerms      = `DELETE FROM initiative_program_info_terms WHERE initiative_id = $1`
	deleteProgramSkills     = `DELETE FROM initiative_program_info_skills WHERE initiative_id = $1`
	deleteProgramConfig     = `DELETE FROM initiative_program_info_config WHERE initiative_id = $1`
	deleteProgramCustomTerm = `DELETE FROM initiative_program_info_custom_term WHERE initiative_id = $1`
	deleteSponsorshipTiers  = `DELETE FROM initiative_sponsorship_tiers WHERE initiative_id = $1`
	deleteOSTIFDetail       = `DELETE FROM initiative_ostif_detail WHERE initiative_id = $1`
	deleteContacts          = `DELETE FROM initiative_contacts WHERE initiative_id = $1`
	deleteEntityDetails     = `DELETE FROM initiative_entity_details WHERE initiative_id = $1`

	updateInitiative = `
		UPDATE initiatives SET
		    name              = $2,
		    slug              = $3,
		    status            = $4,
		    industry          = $5,
		    description       = $6,
		    color             = $7,
		    logo_url          = $8,
		    website_url       = $9,
		    coc_url           = $10,
		    accept_funding    = $11,
		    eventbrite_url    = $12,
		    application_url   = $13,
		    event_start_date  = $14,
		    event_end_date    = $15,
		    country           = $16,
		    city              = $17,
		    is_online         = $18
		WHERE id = $1`
)

// Create inserts a new initiative and all of its child-table rows in a single
// transaction. If any insert fails the transaction is rolled back and the error
// is returned. On success it re-fetches the row via GetByID so that all
// computed columns (e.g. ledger stats) are correctly zero-initialised.
func (r *InitiativeRepository) Create(ctx context.Context, i *models.Initiative, input models.InitiativeCreateInput) (_ *models.Initiative, retErr error) {
	ctx, span := initiativeTracer.Start(ctx, "db.initiatives.Create")
	defer func() {
		if retErr != nil {
			span.RecordError(retErr)
		}
		span.End()
	}()
	span.SetAttributes(attribute.String("db.initiative_id", i.ID))

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// 1. Main initiatives row — includes entity-only fields (NULLed when not applicable).
	if _, err = tx.Exec(ctx, insertInitiative,
		i.ID, i.InitiativeType, i.OwnerID, i.Name, nullableString(i.Slug), nullableString(string(i.Status)),
		nullableString(i.Industry), nullableString(i.Description), nullableString(i.Color),
		nullableString(i.LogoURL), nullableString(i.WebsiteURL), nullableString(i.CocURL),
		nullableString(i.StripePlanID), nullableString(i.StripeProductID),
		i.AcceptFunding,
		nullableString(i.EventbriteURL), nullableString(i.ApplicationURL),
		i.EventStartDate, i.EventEndDate,
		nullableString(i.Country), nullableString(i.City), i.IsOnline,
	); err != nil {
		return nil, fmt.Errorf("create initiative: %w", err)
	}

	// 2. Goals
	for _, g := range input.Goals {
		if _, err = tx.Exec(ctx, insertGoal,
			i.ID, g.Name, g.AmountInCents,
			nullableString(g.Allocation), nullableString(g.RepoLink), nullableString(g.Description),
			nullableString(g.Color), nullableString(g.Icon), g.SortOrder,
		); err != nil {
			return nil, fmt.Errorf("insert goal %q: %w", g.Name, err)
		}
	}

	// 3. Beneficiaries
	for _, b := range input.Beneficiaries {
		if _, err = tx.Exec(ctx, insertBeneficiary,
			i.ID, nullableString(b.Name), nullableString(b.Email),
		); err != nil {
			return nil, fmt.Errorf("insert beneficiary %q: %w", b.Email, err)
		}
	}

	// 4. Custom websites
	for _, w := range input.CustomWebsites {
		if _, err = tx.Exec(ctx, insertCustomWebsite,
			i.ID, nullableString(w.Name), w.URL,
		); err != nil {
			return nil, fmt.Errorf("insert custom website %q: %w", w.URL, err)
		}
	}

	// 5. Contributors (project only)
	for _, c := range input.Contributors {
		if _, err = tx.Exec(ctx, insertContributor,
			i.ID, nullableString(c.Name), nullableString(c.Email),
		); err != nil {
			return nil, fmt.Errorf("insert contributor %q: %w", c.Email, err)
		}
	}

	// 6. Mentors (mentorship only)
	for _, m := range input.Mentors {
		if _, err = tx.Exec(ctx, insertMentor,
			i.ID, nullableString(m.Name), nullableString(m.Email),
			nullableString(m.AvatarURL), nullableString(m.Introduction),
		); err != nil {
			return nil, fmt.Errorf("insert mentor %q: %w", m.Email, err)
		}
	}

	// 7. Program info (mentorship only)
	if input.ProgramInfo != nil {
		for idx, term := range input.ProgramInfo.Terms {
			if _, err = tx.Exec(ctx, insertProgramTerm,
				i.ID, term, idx,
			); err != nil {
				return nil, fmt.Errorf("insert program term %q: %w", term, err)
			}
		}

		for _, skill := range input.ProgramInfo.Skills {
			if _, err = tx.Exec(ctx, insertProgramSkill,
				i.ID, skill,
			); err != nil {
				return nil, fmt.Errorf("insert program skill %q: %w", skill, err)
			}
		}

		if _, err = tx.Exec(ctx, insertProgramConfig,
			i.ID, input.ProgramInfo.TermsConditions,
		); err != nil {
			return nil, fmt.Errorf("insert program config: %w", err)
		}

		if ct := input.ProgramInfo.CustomTerm; ct != nil && ct.TermName != "" {
			if _, err = tx.Exec(ctx, insertProgramCustomTerm,
				i.ID, ct.TermName,
				nullableString(ct.StartMonth), nullableString(ct.EndMonth),
				ct.Year,
			); err != nil {
				return nil, fmt.Errorf("insert program custom term: %w", err)
			}
		}
	}

	// 8. Sponsorship tiers (entity only)
	for _, t := range input.SponsorshipTiers {
		if _, err = tx.Exec(ctx, insertSponsorshipTier,
			i.ID, nullableString(t.Name), nullableString(t.Description),
			nullableString(t.Color), nullableString(t.Icon),
			t.Minimum, t.SortOrder,
		); err != nil {
			return nil, fmt.Errorf("insert sponsorship tier %q: %w", t.Name, err)
		}
	}

	// 9. OSTIF detail (ostif only)
	if input.OSTIFDetail != nil {
		d := input.OSTIFDetail
		if _, err = tx.Exec(ctx, insertOSTIFDetail,
			i.ID,
			nullableString(d.MonetizationStrategy), nullableString(d.CurrentSecurityStrategy),
			nullableString(d.LicenseType), d.TotalBudgetInCents, d.TermsConditions,
		); err != nil {
			return nil, fmt.Errorf("insert ostif detail: %w", err)
		}
	}

	// 10. Contacts (ostif only)
	for _, c := range input.Contacts {
		if _, err = tx.Exec(ctx, insertContact,
			i.ID, c.ContactType,
			nullableString(c.FirstName), nullableString(c.LastName), nullableString(c.Email),
			nullableString(c.PhoneNumber), nullableString(c.OtherContactOption),
			nullableString(c.PreferredContactMethod),
		); err != nil {
			return nil, fmt.Errorf("insert contact %q: %w", c.ContactType, err)
		}
	}

	// 11. GitHub stats (project only — initialised at zero; updated by the sync cron).
	if i.InitiativeType == "project" {
		if _, err = tx.Exec(ctx, insertGitHubStats, i.ID); err != nil {
			return nil, fmt.Errorf("insert github stats: %w", err)
		}
	}

	// 12. Entity details (entity only)
	if len(input.EntityDetails) > 0 {
		detailsJSON, jsonErr := json.Marshal(input.EntityDetails)
		if jsonErr != nil {
			return nil, fmt.Errorf("marshal entity details: %w", jsonErr)
		}
		if _, err = tx.Exec(ctx, insertEntityDetails,
			i.ID, detailsJSON,
		); err != nil {
			return nil, fmt.Errorf("insert entity details: %w", err)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit initiative create: %w", err)
	}
	return r.GetByID(ctx, i.ID)
}

// Update applies changes to an existing initiative and its child-table rows in a single
// transaction. For each child collection a nil value means "leave unchanged"; a non-nil
// value (even empty) replaces all existing rows with the provided set.
func (r *InitiativeRepository) Update(ctx context.Context, i *models.Initiative, input models.InitiativeUpdateInput) (_ *models.Initiative, retErr error) {
	ctx, span := initiativeTracer.Start(ctx, "db.initiatives.Update")
	defer func() {
		if retErr != nil {
			span.RecordError(retErr)
		}
		span.End()
	}()
	span.SetAttributes(attribute.String("db.initiative_id", i.ID))

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// 1. Main initiatives row — all scalar and entity-only fields are always written.
	tag, err := tx.Exec(ctx, updateInitiative,
		i.ID, i.Name, nullableString(i.Slug), nullableString(string(i.Status)),
		nullableString(i.Industry), nullableString(i.Description), nullableString(i.Color),
		nullableString(i.LogoURL), nullableString(i.WebsiteURL), nullableString(i.CocURL),
		i.AcceptFunding,
		nullableString(i.EventbriteURL), nullableString(i.ApplicationURL),
		i.EventStartDate, i.EventEndDate,
		nullableString(i.Country), nullableString(i.City), i.IsOnline,
	)
	if err != nil {
		return nil, fmt.Errorf("update initiative: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, domain.ErrInitiativeNotFound
	}

	// 2. Goals — nil means no-op; non-nil (even empty) replaces all rows.
	if input.Goals != nil {
		if _, err = tx.Exec(ctx, deleteGoals, i.ID); err != nil {
			return nil, fmt.Errorf("delete goals: %w", err)
		}
		for _, g := range input.Goals {
			if _, err = tx.Exec(ctx, insertGoal,
				i.ID, g.Name, g.AmountInCents,
				nullableString(g.Allocation), nullableString(g.RepoLink), nullableString(g.Description),
				nullableString(g.Color), nullableString(g.Icon), g.SortOrder,
			); err != nil {
				return nil, fmt.Errorf("insert goal %q: %w", g.Name, err)
			}
		}
	}

	// 3. Beneficiaries
	if input.Beneficiaries != nil {
		if _, err = tx.Exec(ctx, deleteBeneficiaries, i.ID); err != nil {
			return nil, fmt.Errorf("delete beneficiaries: %w", err)
		}
		for _, b := range input.Beneficiaries {
			if _, err = tx.Exec(ctx, insertBeneficiary,
				i.ID, nullableString(b.Name), nullableString(b.Email),
			); err != nil {
				return nil, fmt.Errorf("insert beneficiary %q: %w", b.Email, err)
			}
		}
	}

	// 4. Custom websites
	if input.CustomWebsites != nil {
		if _, err = tx.Exec(ctx, deleteCustomWebsites, i.ID); err != nil {
			return nil, fmt.Errorf("delete custom websites: %w", err)
		}
		for _, w := range input.CustomWebsites {
			if _, err = tx.Exec(ctx, insertCustomWebsite,
				i.ID, nullableString(w.Name), w.URL,
			); err != nil {
				return nil, fmt.Errorf("insert custom website %q: %w", w.URL, err)
			}
		}
	}

	// 5. Contributors
	if input.Contributors != nil {
		if _, err = tx.Exec(ctx, deleteContributors, i.ID); err != nil {
			return nil, fmt.Errorf("delete contributors: %w", err)
		}
		for _, c := range input.Contributors {
			if _, err = tx.Exec(ctx, insertContributor,
				i.ID, nullableString(c.Name), nullableString(c.Email),
			); err != nil {
				return nil, fmt.Errorf("insert contributor %q: %w", c.Email, err)
			}
		}
	}

	// 6. Mentors
	if input.Mentors != nil {
		if _, err = tx.Exec(ctx, deleteMentors, i.ID); err != nil {
			return nil, fmt.Errorf("delete mentors: %w", err)
		}
		for _, m := range input.Mentors {
			if _, err = tx.Exec(ctx, insertMentor,
				i.ID, nullableString(m.Name), nullableString(m.Email),
				nullableString(m.AvatarURL), nullableString(m.Introduction),
			); err != nil {
				return nil, fmt.Errorf("insert mentor %q: %w", m.Email, err)
			}
		}
	}

	// 7. Program info — nil pointer = no-op; non-nil = replace all four sub-tables.
	if input.ProgramInfo != nil {
		for _, q := range []string{deleteProgramTerms, deleteProgramSkills, deleteProgramConfig, deleteProgramCustomTerm} {
			if _, err = tx.Exec(ctx, q, i.ID); err != nil {
				return nil, fmt.Errorf("delete program info: %w", err)
			}
		}
		for idx, term := range input.ProgramInfo.Terms {
			if _, err = tx.Exec(ctx, insertProgramTerm, i.ID, term, idx); err != nil {
				return nil, fmt.Errorf("insert program term %q: %w", term, err)
			}
		}
		for _, skill := range input.ProgramInfo.Skills {
			if _, err = tx.Exec(ctx, insertProgramSkill, i.ID, skill); err != nil {
				return nil, fmt.Errorf("insert program skill %q: %w", skill, err)
			}
		}
		if _, err = tx.Exec(ctx, insertProgramConfig, i.ID, input.ProgramInfo.TermsConditions); err != nil {
			return nil, fmt.Errorf("insert program config: %w", err)
		}
		if ct := input.ProgramInfo.CustomTerm; ct != nil && ct.TermName != "" {
			if _, err = tx.Exec(ctx, insertProgramCustomTerm,
				i.ID, ct.TermName,
				nullableString(ct.StartMonth), nullableString(ct.EndMonth),
				ct.Year,
			); err != nil {
				return nil, fmt.Errorf("insert program custom term: %w", err)
			}
		}
	}

	// 8. Sponsorship tiers
	if input.SponsorshipTiers != nil {
		if _, err = tx.Exec(ctx, deleteSponsorshipTiers, i.ID); err != nil {
			return nil, fmt.Errorf("delete sponsorship tiers: %w", err)
		}
		for _, t := range input.SponsorshipTiers {
			if _, err = tx.Exec(ctx, insertSponsorshipTier,
				i.ID, nullableString(t.Name), nullableString(t.Description),
				nullableString(t.Color), nullableString(t.Icon),
				t.Minimum, t.SortOrder,
			); err != nil {
				return nil, fmt.Errorf("insert sponsorship tier %q: %w", t.Name, err)
			}
		}
	}

	// 9. OSTIF detail — nil pointer = no-op; non-nil = replace.
	if input.OSTIFDetail != nil {
		if _, err = tx.Exec(ctx, deleteOSTIFDetail, i.ID); err != nil {
			return nil, fmt.Errorf("delete ostif detail: %w", err)
		}
		d := input.OSTIFDetail
		if _, err = tx.Exec(ctx, insertOSTIFDetail,
			i.ID,
			nullableString(d.MonetizationStrategy), nullableString(d.CurrentSecurityStrategy),
			nullableString(d.LicenseType), d.TotalBudgetInCents, d.TermsConditions,
		); err != nil {
			return nil, fmt.Errorf("insert ostif detail: %w", err)
		}
	}

	// 10. Contacts
	if input.Contacts != nil {
		if _, err = tx.Exec(ctx, deleteContacts, i.ID); err != nil {
			return nil, fmt.Errorf("delete contacts: %w", err)
		}
		for _, c := range input.Contacts {
			if _, err = tx.Exec(ctx, insertContact,
				i.ID, c.ContactType,
				nullableString(c.FirstName), nullableString(c.LastName), nullableString(c.Email),
				nullableString(c.PhoneNumber), nullableString(c.OtherContactOption),
				nullableString(c.PreferredContactMethod),
			); err != nil {
				return nil, fmt.Errorf("insert contact %q: %w", c.ContactType, err)
			}
		}
	}

	// 11. Entity details — nil map = no-op; non-nil (even empty) = replace.
	if input.EntityDetails != nil {
		if _, err = tx.Exec(ctx, deleteEntityDetails, i.ID); err != nil {
			return nil, fmt.Errorf("delete entity details: %w", err)
		}
		if len(input.EntityDetails) > 0 {
			detailsJSON, jsonErr := json.Marshal(input.EntityDetails)
			if jsonErr != nil {
				return nil, fmt.Errorf("marshal entity details: %w", jsonErr)
			}
			if _, err = tx.Exec(ctx, insertEntityDetails, i.ID, detailsJSON); err != nil {
				return nil, fmt.Errorf("insert entity details: %w", err)
			}
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit initiative update: %w", err)
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
	defer rows.Close() //nolint:errcheck
	return scanGoals(rows)
}

// listGoalsForIDs fetches goals for a set of initiative IDs in one query,
// returning a map of initiativeID → []Goal.
func (r *InitiativeRepository) listGoalsForIDs(ctx context.Context, ids []string) (map[string][]models.Goal, error) {
	// Build $1,$2,... placeholders using a Builder to avoid quadratic string copies.
	args := make([]any, len(ids))
	var sb strings.Builder
	for i, id := range ids {
		if i > 0 {
			sb.WriteByte(',')
		}
		fmt.Fprintf(&sb, "$%d", i+1)
		args[i] = id
	}

	q := fmt.Sprintf(`
		SELECT id, initiative_id, name, amount_in_cents, allocation,
		       repo_link, description, color, icon, sort_order,
		       created_on, updated_on
		FROM initiative_goals
		WHERE initiative_id IN (%s)
		ORDER BY initiative_id, sort_order ASC`, sb.String())

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list goals for ids: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	result := make(map[string][]models.Goal)
	for rows.Next() {
		g, err := scanGoal(rows)
		if err != nil {
			return nil, fmt.Errorf("scan goal: %w", err)
		}
		result[g.InitiativeID] = append(result[g.InitiativeID], g)
	}
	return result, rows.Err()
}

func scanGoals(rows pgx.Rows) ([]models.Goal, error) {
	goals := []models.Goal{}
	for rows.Next() {
		g, err := scanGoal(rows)
		if err != nil {
			return nil, fmt.Errorf("scan goal: %w", err)
		}
		goals = append(goals, g)
	}
	return goals, rows.Err()
}

func scanGoal(row scanner) (models.Goal, error) {
	var g models.Goal
	var allocation, repoLink, description, color, icon *string
	err := row.Scan(
		&g.ID, &g.InitiativeID, &g.Name, &g.AmountInCents, &allocation,
		&repoLink, &description, &color, &icon, &g.SortOrder,
		&g.CreatedOn, &g.UpdatedOn,
	)
	if err != nil {
		return g, err
	}
	g.Allocation = derefString(allocation)
	g.RepoLink = derefString(repoLink)
	g.Description = derefString(description)
	g.Color = derefString(color)
	g.Icon = derefString(icon)
	return g, nil
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
		sponsorsJSON                                                  []byte
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
		&sponsorsJSON,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrInitiativeNotFound
		}
		return nil, err
	}

	if len(sponsorsJSON) > 0 {
		if err := json.Unmarshal(sponsorsJSON, &i.RawSponsors); err != nil {
			slog.Warn("failed to unmarshal sponsors JSONB", "initiative_id", i.ID, "error", err)
		}
	}

	i.SourceDynamoTable = derefString(sourceDynamoTable)
	i.Slug = derefString(slug)
	i.Status = models.InitiativeStatus(strings.ToLower(derefString(status)))
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

// GetUsersByIDs returns a map of Auth0 user_id → User for all IDs provided.
// Missing IDs are absent from the map.
func (r *InitiativeRepository) GetUsersByIDs(ctx context.Context, userIDs []string) (map[string]models.User, error) {
	ctx, span := initiativeTracer.Start(ctx, "db.initiative.GetUsersByIDs")
	defer span.End()

	result := make(map[string]models.User, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}

	const q = `SELECT id, user_id, name, avatar_url FROM users WHERE user_id = ANY($1)`
	rows, err := r.pool.Query(ctx, q, userIDs)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("get users by IDs: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	for rows.Next() {
		var u models.User
		var name, avatarURL *string
		if err := rows.Scan(&u.ID, &u.UserID, &name, &avatarURL); err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("scan user: %w", err)
		}
		if name != nil {
			u.Name = *name
		}
		if avatarURL != nil {
			u.AvatarURL = *avatarURL
		}
		result[u.UserID] = u
	}
	if err := rows.Err(); err != nil {
		span.RecordError(err)
		return result, fmt.Errorf("iterate users: %w", err)
	}
	return result, nil
}

// GetOrganizationsByIDs returns a map of org UUID → Organization for all IDs provided.
// Missing IDs are absent from the map.
func (r *InitiativeRepository) GetOrganizationsByIDs(ctx context.Context, ids []string) (map[string]models.Organization, error) {
	ctx, span := initiativeTracer.Start(ctx, "db.initiative.GetOrganizationsByIDs")
	defer span.End()

	result := make(map[string]models.Organization, len(ids))
	if len(ids) == 0 {
		return result, nil
	}

	const q = `SELECT id, name, avatar_url FROM organizations WHERE id = ANY($1::uuid[])`
	rows, err := r.pool.Query(ctx, q, ids)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("get organizations by IDs: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	for rows.Next() {
		var o models.Organization
		var avatarURL *string
		if err := rows.Scan(&o.ID, &o.Name, &avatarURL); err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("scan organization: %w", err)
		}
		if avatarURL != nil {
			o.AvatarURL = *avatarURL
		}
		result[o.ID] = o
	}
	if err := rows.Err(); err != nil {
		span.RecordError(err)
		return result, fmt.Errorf("iterate organizations: %w", err)
	}
	return result, nil
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
