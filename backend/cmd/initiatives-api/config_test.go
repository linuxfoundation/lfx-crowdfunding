// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package main

import (
	"strings"
	"testing"
)

func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://crowdfunding:crowdfunding@localhost:5432/crowdfunding")
	t.Setenv("HEIMDALL_JWT_AUDIENCE", "lfx-crowdfunding-backend")
	t.Setenv("HEIMDALL_JWT_ISSUER", "heimdall")
	t.Setenv("STRIPE_SECRET_KEY", "sk_test_123")
	t.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_123")
	t.Setenv("LEDGER_BASE_URL", "https://ledger.example")
	t.Setenv("LEDGER_API_KEY", "ledger-key")
	t.Setenv("STRIPE_RETURN_URL", "https://frontend.example/payment/complete")
	t.Setenv("FRONTEND_BASE_URL", "https://frontend.example")
	t.Setenv("S3_UPLOAD_BUCKET", "crowdfunding-uploads-test")
	t.Setenv("HEIMDALL_JWKS_URL", "http://heimdall.example:4457/.well-known/jwks")
	t.Setenv("DISABLED_MOCK_LOCAL_PRINCIPAL", "")
	t.Setenv("ALLOW_MOCK_LOCAL_PRINCIPAL_BYPASS", "false")
}

func TestLoadConfig_BypassPrincipalRequiresAllowFlag(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("HEIMDALL_JWKS_URL", "")
	t.Setenv("DISABLED_MOCK_LOCAL_PRINCIPAL", "local-dev-user")
	t.Setenv("ALLOW_MOCK_LOCAL_PRINCIPAL_BYPASS", "false")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error when bypass principal is set without explicit allow flag")
	}
	if !strings.Contains(err.Error(), "ALLOW_MOCK_LOCAL_PRINCIPAL_BYPASS=true") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadConfig_BypassPrincipalAllowedWithExplicitFlag(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("HEIMDALL_JWKS_URL", "")
	t.Setenv("DISABLED_MOCK_LOCAL_PRINCIPAL", "local-dev-user")
	t.Setenv("ALLOW_MOCK_LOCAL_PRINCIPAL_BYPASS", "true")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if !cfg.Local.AllowMockLocalPrincipalBypass {
		t.Fatal("expected AllowMockLocalPrincipalBypass to be true")
	}
	if cfg.Local.DisabledMockLocalPrincipal != "local-dev-user" {
		t.Fatalf("DisabledMockLocalPrincipal = %q, want %q", cfg.Local.DisabledMockLocalPrincipal, "local-dev-user")
	}
}

func TestLoadConfig_WhitespaceBypassPrincipalIsIgnored(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("DISABLED_MOCK_LOCAL_PRINCIPAL", "   \t  ")
	t.Setenv("ALLOW_MOCK_LOCAL_PRINCIPAL_BYPASS", "false")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.Local.DisabledMockLocalPrincipal != "" {
		t.Fatalf("DisabledMockLocalPrincipal = %q, want empty string", cfg.Local.DisabledMockLocalPrincipal)
	}
}

func TestLoadConfig_HeimdallAudienceAndIssuerRequired(t *testing.T) {
	for _, name := range []string{"HEIMDALL_JWT_AUDIENCE", "HEIMDALL_JWT_ISSUER"} {
		t.Run(name, func(t *testing.T) {
			setRequiredEnv(t)
			t.Setenv(name, "")

			_, err := LoadConfig()
			if err == nil {
				t.Fatalf("expected error when %s is empty", name)
			}
			if !strings.Contains(err.Error(), "HEIMDALL_JWT_AUDIENCE and HEIMDALL_JWT_ISSUER are required") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
