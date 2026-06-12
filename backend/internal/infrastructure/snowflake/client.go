// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package snowflake

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"fmt"

	"github.com/snowflakedb/gosnowflake"
	_ "github.com/snowflakedb/gosnowflake"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

const fetchProgramsQuery = `
SELECT
	p.jobspring_project_id,
	p.name,
	p.status,
	COALESCE(p.mentee_goal_cents, 0) AS mentee_goal_cents
FROM ANALYTICS.GOLD_FACT.MENTORSHIP_PROGRAMS p
WHERE p.jobspring_project_id IS NOT NULL
`

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
	cfg_snowflake := &gosnowflake.Config{
		Account:    cfg.Account,
		User:       cfg.User,
		Warehouse:  cfg.Warehouse,
		Database:   cfg.Database,
		Role:       cfg.Role,
		PrivateKey: privateKey,
	}

	// Construct DSN.
	dsn, err := gosnowflake.DSN(cfg_snowflake)
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
func (c *Client) FetchPrograms(ctx context.Context) ([]models.MentorshipProgram, error) {
	rows, err := c.db.QueryContext(ctx, fetchProgramsQuery)
	if err != nil {
		return nil, fmt.Errorf("query mentorship programs: %w", err)
	}
	defer rows.Close()

	var programs []models.MentorshipProgram
	for rows.Next() {
		var p models.MentorshipProgram
		if err := rows.Scan(
			&p.JobspringProjectID,
			&p.Name,
			&p.Status,
			&p.MenteeGoalCents,
		); err != nil {
			return nil, fmt.Errorf("scan mentorship program row: %w", err)
		}
		programs = append(programs, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate mentorship program rows: %w", err)
	}
	return programs, nil
}
