// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package main is the entrypoint for the LFX Crowdfunding Initiatives API.
package main

import (
	"fmt"
	"os"
	"strconv"
	"time"
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
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// JWTConfig holds Auth0 / JWKS settings.
type JWTConfig struct {
	JWKSURL  string
	Audience string
	Issuer   string
}

// StripeConfig holds Stripe API settings.
type StripeConfig struct {
	SecretKey     string
	WebhookSecret string
	Timeout       time.Duration
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

	return &Config{
		Server: ServerConfig{
			Port:            getIntEnv("PORT", 8080),
			ReadTimeout:     getDurationEnv("SERVER_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:    getDurationEnv("SERVER_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:     getDurationEnv("SERVER_IDLE_TIMEOUT", 60*time.Second),
			ShutdownTimeout: getDurationEnv("SERVER_SHUTDOWN_TIMEOUT", 30*time.Second),
		},
		Database: DatabaseConfig{
			DSN:             dsn,
			MaxOpenConns:    getIntEnv("DB_MAX_CONNS", 20),
			MaxIdleConns:    getIntEnv("DB_MIN_CONNS", 2),
			ConnMaxLifetime: getDurationEnv("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		JWT: JWTConfig{
			JWKSURL:  jwksURL,
			Audience: getEnv("JWT_AUDIENCE", ""),
			Issuer:   getEnv("JWT_ISSUER", ""),
		},
		Stripe: StripeConfig{
			SecretKey:     stripeKey,
			WebhookSecret: stripeWebhookSecret,
			Timeout:       getDurationEnv("STRIPE_TIMEOUT", 30*time.Second),
		},
		Ledger: LedgerConfig{
			BaseURL: ledgerBaseURL,
			APIKey:  getEnv("LEDGER_API_KEY", ""),
			Timeout: getDurationEnv("LEDGER_TIMEOUT", 10*time.Second),
		},
		OTel: OTelConfig{
			ServiceName:    getEnv("OTEL_SERVICE_NAME", "lfx-v2-initiatives-service"),
			ServiceVersion: getEnv("OTEL_SERVICE_VERSION", "dev"),
			Endpoint:       getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
		},
		Local: LocalConfig{
			DisabledMockLocalPrincipal: getEnv("DISABLED_MOCK_LOCAL_PRINCIPAL", ""),
		},
	}, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func getIntEnv(key string, fallback int) int {
	v := getEnv(key, "")
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	v := getEnv(key, "")
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

func getBoolEnv(key string, fallback bool) bool {
	v := getEnv(key, "")
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}
