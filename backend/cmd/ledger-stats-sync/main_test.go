// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package main

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfig_missingDatabaseURL(t *testing.T) {
	clearEnv(t)
	_, err := loadConfig()
	if err == nil {
		t.Fatal("expected error for missing DATABASE_URL, got nil")
	}
}

func TestLoadConfig_missingLedgerBaseURL(t *testing.T) {
	clearEnv(t)
	t.Setenv("DATABASE_URL", "postgres://localhost/test")

	_, err := loadConfig()
	if err == nil {
		t.Fatal("expected error for missing LEDGER_BASE_URL, got nil")
	}
}

func TestLoadConfig_missingLedgerAPIKey(t *testing.T) {
	clearEnv(t)
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("LEDGER_BASE_URL", "https://ledger.example.com")

	_, err := loadConfig()
	if err == nil {
		t.Fatal("expected error for missing LEDGER_API_KEY, got nil")
	}
}

func TestLoadConfig_invalidLedgerTimeout(t *testing.T) {
	clearEnv(t)
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("LEDGER_BASE_URL", "https://ledger.example.com")
	t.Setenv("LEDGER_API_KEY", "Bearer token123")
	t.Setenv("LEDGER_TIMEOUT", "not-a-duration")

	_, err := loadConfig()
	if err == nil {
		t.Fatal("expected error for invalid LEDGER_TIMEOUT, got nil")
	}
}

func TestLoadConfig_defaultLedgerTimeout(t *testing.T) {
	clearEnv(t)
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("LEDGER_BASE_URL", "https://ledger.example.com")
	t.Setenv("LEDGER_API_KEY", "Bearer token123")

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.LedgerTimeout != 30*time.Second {
		t.Errorf("LedgerTimeout: got %v, want 30s", cfg.LedgerTimeout)
	}
}

func TestLoadConfig_customLedgerTimeout(t *testing.T) {
	clearEnv(t)
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("LEDGER_BASE_URL", "https://ledger.example.com")
	t.Setenv("LEDGER_API_KEY", "Bearer token123")
	t.Setenv("LEDGER_TIMEOUT", "15s")

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.LedgerTimeout != 15*time.Second {
		t.Errorf("LedgerTimeout: got %v, want 15s", cfg.LedgerTimeout)
	}
}

func TestLoadConfig_allRequiredFieldsPopulated(t *testing.T) {
	clearEnv(t)
	t.Setenv("DATABASE_URL", "postgres://localhost/crowdfunding")
	t.Setenv("LEDGER_BASE_URL", "https://ledger.example.com/")
	t.Setenv("LEDGER_API_KEY", "Bearer secret")

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DatabaseURL != "postgres://localhost/crowdfunding" {
		t.Errorf("DatabaseURL: got %q", cfg.DatabaseURL)
	}
	if cfg.LedgerBaseURL != "https://ledger.example.com/" {
		t.Errorf("LedgerBaseURL: got %q", cfg.LedgerBaseURL)
	}
	if cfg.LedgerAPIKey != "Bearer secret" {
		t.Errorf("LedgerAPIKey: got %q", cfg.LedgerAPIKey)
	}
}

// clearEnv unsets all environment variables read by loadConfig and registers a
// cleanup to restore them after the test.
func clearEnv(t *testing.T) {
	t.Helper()
	vars := []string{"DATABASE_URL", "LEDGER_BASE_URL", "LEDGER_API_KEY", "LEDGER_TIMEOUT"}
	for _, v := range vars {
		old, exists := os.LookupEnv(v)
		os.Unsetenv(v) //nolint:errcheck
		if exists {
			t.Cleanup(func() { os.Setenv(v, old) }) //nolint:errcheck
		}
	}
}
