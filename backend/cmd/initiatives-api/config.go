// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package main is the entrypoint for the LFX Crowdfunding Initiatives API.
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/auth"
)

// Config holds all runtime configuration for the service.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Stripe   StripeConfig
	Ledger   LedgerConfig
	OTel     OTelConfig
	Local    LocalConfig
	Approval ApprovalConfig
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

// DatabaseConfig holds PostgreSQL connection settings.
type DatabaseConfig struct {
	DSN             string
	MaxConns        int
	MinConns        int
	ConnMaxLifetime time.Duration
}

// JWTConfig holds Auth0 / JWKS settings.
type JWTConfig struct {
	JWKSURL   string
	Audience  string
	Issuer    string
	ClockSkew time.Duration
}

// StripeConfig holds Stripe API settings.
type StripeConfig struct {
	SecretKey     string
	WebhookSecret string
	Timeout       time.Duration
	// ReturnURL is the frontend URL Stripe redirects to after a 3DS challenge.
	// Required when Confirm=true on a PaymentIntent. Set STRIPE_RETURN_URL.
	ReturnURL string
	// AckUnimplementedWebhooks, when true, responds with HTTP 200 for
	// recognised-but-unimplemented event types instead of 501. Useful in
	// pre-production environments where real Stripe deliveries are active but
	// DB persistence has not yet landed. Set STRIPE_WEBHOOK_ACK_UNIMPLEMENTED=true.
	AckUnimplementedWebhooks bool
}

// LedgerConfig holds the upstream Ledger service settings.
type LedgerConfig struct {
	BaseURL string
	APIKey  string
	Timeout time.Duration
}

// OTelConfig holds OpenTelemetry settings.
type OTelConfig struct {
	ServiceName    string
	ServiceVersion string
	Endpoint       string
}

// LocalConfig holds development-only settings.
type LocalConfig struct {
	// DisabledMockLocalPrincipal, when non-empty, bypasses JWT validation and
	// injects the value as the mock principal sub. NEVER set in production.
	DisabledMockLocalPrincipal string
}

// ApprovalConfig holds initiative approval settings.
type ApprovalConfig struct {
	// AllowedApprovers is the list of usernames permitted to approve or decline
	// initiatives. Sourced from the ALLOWED_APPROVERS env var (comma-separated).
	AllowedApprovers []string
}

// LoadConfig reads all configuration from environment variables.
func LoadConfig() (*Config, error) {
	dsn := getEnv("DATABASE_URL", "")
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	jwksURL := getEnv("JWKS_URL", "")
	mockPrincipal := getEnv("DISABLED_MOCK_LOCAL_PRINCIPAL", "")
	if jwksURL == "" && mockPrincipal == "" {
		return nil, fmt.Errorf("JWKS_URL is required (or set DISABLED_MOCK_LOCAL_PRINCIPAL for local dev)")
	}
	// Default audience and issuer values keep JWT validation configured even when
	// the corresponding environment variables are not explicitly set. Treat
	// empty-string values as unset as well, since Helm/Kubernetes may inject
	// present-but-empty environment variables from default ConfigMap entries.
	jwtAudience, ok := os.LookupEnv("JWT_AUDIENCE")
	if !ok || jwtAudience == "" {
		jwtAudience = auth.DefaultAudience
	}
	jwtIssuer, ok := os.LookupEnv("JWT_ISSUER")
	if !ok || jwtIssuer == "" {
		jwtIssuer = auth.DefaultIssuer
	}
	stripeKey := getEnv("STRIPE_SECRET_KEY", "")
	if stripeKey == "" {
		return nil, fmt.Errorf("STRIPE_SECRET_KEY is required")
	}
	stripeWebhookSecret := getEnv("STRIPE_WEBHOOK_SECRET", "")
	if stripeWebhookSecret == "" {
		return nil, fmt.Errorf("STRIPE_WEBHOOK_SECRET is required")
	}
	ledgerBaseURL := getEnv("LEDGER_BASE_URL", "")
	if ledgerBaseURL == "" {
		return nil, fmt.Errorf("LEDGER_BASE_URL is required")
	}
	ledgerAPIKey := getEnv("LEDGER_API_KEY", "")
	if ledgerAPIKey == "" {
		return nil, fmt.Errorf("LEDGER_API_KEY is required")
	}

	port, err := getIntEnv("PORT", 8080)
	if err != nil {
		return nil, err
	}
	readTimeout, err := getDurationEnv("SERVER_READ_TIMEOUT", 15*time.Second)
	if err != nil {
		return nil, err
	}
	writeTimeout, err := getDurationEnv("SERVER_WRITE_TIMEOUT", 30*time.Second)
	if err != nil {
		return nil, err
	}
	idleTimeout, err := getDurationEnv("SERVER_IDLE_TIMEOUT", 60*time.Second)
	if err != nil {
		return nil, err
	}
	shutdownTimeout, err := getDurationEnv("SERVER_SHUTDOWN_TIMEOUT", 30*time.Second)
	if err != nil {
		return nil, err
	}
	maxConns, err := getIntEnv("DB_MAX_CONNS", 20)
	if err != nil {
		return nil, err
	}
	minConns, err := getIntEnv("DB_MIN_CONNS", 2)
	if err != nil {
		return nil, err
	}
	connMaxLifetime, err := getDurationEnv("DB_CONN_MAX_LIFETIME", 5*time.Minute)
	if err != nil {
		return nil, err
	}
	stripeTimeout, err := getDurationEnv("STRIPE_TIMEOUT", 30*time.Second)
	if err != nil {
		return nil, err
	}
	stripeAckUnimplemented, err := getBoolEnv("STRIPE_WEBHOOK_ACK_UNIMPLEMENTED", false)
	if err != nil {
		return nil, err
	}
	stripeReturnURL := getEnv("STRIPE_RETURN_URL", "")
	if stripeReturnURL == "" {
		return nil, fmt.Errorf("STRIPE_RETURN_URL is required (set to the frontend URL Stripe redirects to after 3DS, e.g. https://yourdomain.com/payment/complete)")
	}
	ledgerTimeout, err := getDurationEnv("LEDGER_TIMEOUT", 10*time.Second)
	if err != nil {
		return nil, err
	}

	return &Config{
		Server: ServerConfig{
			Port:            port,
			ReadTimeout:     readTimeout,
			WriteTimeout:    writeTimeout,
			IdleTimeout:     idleTimeout,
			ShutdownTimeout: shutdownTimeout,
		},
		Database: DatabaseConfig{
			DSN:             dsn,
			MaxConns:        maxConns,
			MinConns:        minConns,
			ConnMaxLifetime: connMaxLifetime,
		},
		JWT: JWTConfig{
			JWKSURL:   jwksURL,
			Audience:  jwtAudience,
			Issuer:    jwtIssuer,
			ClockSkew: auth.DefaultClockSkew,
		},
		Stripe: StripeConfig{
			SecretKey:                stripeKey,
			WebhookSecret:            stripeWebhookSecret,
			Timeout:                  stripeTimeout,
			ReturnURL:                stripeReturnURL,
			AckUnimplementedWebhooks: stripeAckUnimplemented,
		},
		Ledger: LedgerConfig{
			BaseURL: ledgerBaseURL,
			APIKey:  ledgerAPIKey,
			Timeout: ledgerTimeout,
		},
		OTel: OTelConfig{
			ServiceName:    getEnv("OTEL_SERVICE_NAME", "lfx-v2-initiatives-service"),
			ServiceVersion: getEnv("OTEL_SERVICE_VERSION", "dev"),
			Endpoint:       getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
		},
		Local: LocalConfig{
			DisabledMockLocalPrincipal: getEnv("DISABLED_MOCK_LOCAL_PRINCIPAL", ""),
		},
		Approval: ApprovalConfig{
			AllowedApprovers: parseCommaList(getEnv("ALLOWED_APPROVERS", "")),
		},
	}, nil
}

// parseCommaList splits a comma-separated string into trimmed, non-empty tokens.
func parseCommaList(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func getIntEnv(key string, fallback int) (int, error) {
	v := getEnv(key, "")
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("env %s: invalid integer %q: %w", key, v, err)
	}
	return n, nil
}

func getDurationEnv(key string, fallback time.Duration) (time.Duration, error) {
	v := getEnv(key, "")
	if v == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("env %s: invalid duration %q: %w", key, v, err)
	}
	return d, nil
}

func getBoolEnv(key string, fallback bool) (bool, error) {
	v := getEnv(key, "")
	if v == "" {
		return fallback, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("env %s: invalid boolean %q: %w", key, v, err)
	}
	return b, nil
}
