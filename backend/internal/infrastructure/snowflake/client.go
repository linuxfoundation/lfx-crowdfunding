// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package snowflake

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"strings"
	"time"

	"github.com/snowflakedb/gosnowflake"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

var snowflakeTracer = otel.Tracer("snowflake-client")

const fetchProgramsQuery = `
SELECT
	p.PROGRAM_ID,
	p.PROGRAM_NAME,
	p.PROGRAM_STATUS,
	p.PROGRAM_DESCRIPTION,
	p.program_slug,
	COALESCE(p.OWNER_LF_USERNAME, '') AS OWNER_LF_USERNAME,
	p.PROGRAM_TECHNOLOGY,
	p.SELECTED_MENTEES,
	p.mentors,
	p.program_skills,
	p.UPDATED_AT
FROM ANALYTICS.GOLD_FACT.MENTORSHIP_PROGRAMS p
WHERE p.PROGRAM_ID IS NOT NULL
`

// snowflakePerson is the shared JSON structure for entries in the
// SELECTED_MENTEES and mentors VARIANT array columns.
type snowflakePerson struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

// ClientConfig holds credentials for connecting to Snowflake via key-pair auth.
type ClientConfig struct {
	Account    string
	User       string
	Warehouse  string
	Database   string
	Role       string
	PrivateKey string // PEM-encoded PKCS8 private key
}

// Client queries Snowflake for Mentorship program data.
type Client struct {
	db *sql.DB
}

// NewClient opens a Snowflake connection using key-pair authentication.
func NewClient(cfg ClientConfig) (*Client, error) {
	// Parse the PEM-encoded private key.
	block, rest := pem.Decode([]byte(cfg.PrivateKey))
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM private key: no PEM block found")
	}
	if len(rest) > 0 {
		return nil, fmt.Errorf("failed to parse PEM private key: extra data after PEM block")
	}

	// Parse the PKCS8 DER-encoded private key.
	privateKeyInterface, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PKCS8 private key: %w", err)
	}

	privateKey, ok := privateKeyInterface.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key is not RSA")
	}

	// Build Snowflake connection config.
	sfCfg := &gosnowflake.Config{
		Account:       cfg.Account,
		User:          cfg.User,
		Warehouse:     cfg.Warehouse,
		Database:      cfg.Database,
		Role:          cfg.Role,
		Authenticator: gosnowflake.AuthTypeJwt,
		PrivateKey:    privateKey,
	}

	// Construct DSN.
	dsn, err := gosnowflake.DSN(sfCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to construct DSN: %w", err)
	}

	// Open database connection.
	db, err := sql.Open("snowflake", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open snowflake connection: %w", err)
	}

	return &Client{db: db}, nil
}

// NewClientFromDB constructs a Client from an existing *sql.DB.
// Used in tests to inject a mock driver.
func NewClientFromDB(db *sql.DB) *Client {
	return &Client{db: db}
}

// Close releases the underlying database connection pool.
func (c *Client) Close() error {
	return c.db.Close()
}

// FetchPrograms runs the Snowflake query and returns all Mentorship programs.
// VARIANT columns (SELECTED_MENTEES, mentors, program_skills) are parsed from
// their JSON string representation. A NULL VARIANT column means the field is
// absent and the corresponding slice on the model is left nil (skip upsert).
func (c *Client) FetchPrograms(ctx context.Context) (_ []models.MentorshipProgram, retErr error) {
	ctx, span := snowflakeTracer.Start(ctx, "FetchPrograms")
	defer func() {
		if retErr != nil {
			span.RecordError(retErr)
			span.SetStatus(codes.Error, retErr.Error())
		}
		span.End()
	}()

	rows, err := c.db.QueryContext(ctx, fetchProgramsQuery)
	if err != nil {
		return nil, fmt.Errorf("query mentorship programs: %w", err)
	}
	defer rows.Close()

	var programs []models.MentorshipProgram
	for rows.Next() {
		var (
			programID       string
			name            string
			status          string
			description     sql.NullString
			slug            sql.NullString
			ownerLFUsername string
			industry        sql.NullString
			menteesJSON     sql.NullString
			mentorsJSON     sql.NullString
			skillsJSON      sql.NullString
			_updatedAt      time.Time // only used in the WHERE clause; not persisted
		)
		if err := rows.Scan(
			&programID,
			&name,
			&status,
			&description,
			&slug,
			&ownerLFUsername,
			&industry,
			&menteesJSON,
			&mentorsJSON,
			&skillsJSON,
			&_updatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan mentorship program row: %w", err)
		}

		p := models.MentorshipProgram{
			JobspringProjectID: programID,
			Name:               name,
			Status:             status,
			Description:        description.String,
			Slug:               slug.String,
			OwnerLFUsername:    ownerLFUsername,
			Industry:           industry.String,
		}

		// Parse VARIANT JSON columns. A NULL column → nil slice (skip upsert).
		if menteesJSON.Valid && menteesJSON.String != "" {
			var err error
			if p.Beneficiaries, err = parseMentees(menteesJSON.String); err != nil {
				return nil, fmt.Errorf("parse SELECTED_MENTEES for %q: %w", programID, err)
			}
		}
		if mentorsJSON.Valid && mentorsJSON.String != "" {
			var err error
			if p.Mentors, err = parseMentors(mentorsJSON.String); err != nil {
				return nil, fmt.Errorf("parse mentors for %q: %w", programID, err)
			}
		}
		if skillsJSON.Valid && skillsJSON.String != "" {
			var err error
			if p.Skills, err = parseSkills(skillsJSON.String); err != nil {
				return nil, fmt.Errorf("parse program_skills for %q: %w", programID, err)
			}
		}

		programs = append(programs, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate mentorship program rows: %w", err)
	}
	return programs, nil
}

// parseMentees decodes the SELECTED_MENTEES VARIANT JSON array into beneficiary models.
func parseMentees(js string) ([]models.MentorshipBeneficiary, error) {
	var raw []snowflakePerson
	if err := json.Unmarshal([]byte(js), &raw); err != nil {
		return nil, err
	}
	out := make([]models.MentorshipBeneficiary, 0, len(raw))
	for _, p := range raw {
		out = append(out, models.MentorshipBeneficiary{
			Name:  strings.TrimSpace(p.FirstName + " " + p.LastName),
			Email: p.Email,
		})
	}
	return out, nil
}

// parseMentors decodes the mentors VARIANT JSON array into mentor models.
func parseMentors(js string) ([]models.MentorshipMentor, error) {
	var raw []snowflakePerson
	if err := json.Unmarshal([]byte(js), &raw); err != nil {
		return nil, err
	}
	out := make([]models.MentorshipMentor, 0, len(raw))
	for _, p := range raw {
		out = append(out, models.MentorshipMentor{
			Name:      strings.TrimSpace(p.FirstName + " " + p.LastName),
			Email:     p.Email,
			AvatarURL: p.AvatarURL,
		})
	}
	return out, nil
}

// parseSkills decodes the program_skills VARIANT JSON array into a string slice.
func parseSkills(js string) ([]string, error) {
	var raw []string
	if err := json.Unmarshal([]byte(js), &raw); err != nil {
		return nil, err
	}
	return raw, nil
}
