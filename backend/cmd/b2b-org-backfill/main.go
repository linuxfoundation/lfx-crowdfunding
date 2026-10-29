// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// b2b-org-backfill attaches b2b_org SFIDs to org donations and subscriptions
// that don't have one (lfx-crowdfunding#412): rows saved before the lookup on
// save existed (#411), and rows whose lookup on save failed or found no match.
//
// It looks up each distinct unmatched org_domain once with member-service's
// lfx.member.b2b_org_lookup_by_website, then fills the domain's donations and
// subscriptions. Live subscriptions also get b2b_org_sfid in their Stripe
// metadata, so ledger-service's hook carries it on renewals. Only empty SFIDs
// are filled, so it's safe to re-run; it's deployed as the b2b-org-backfill
// CronJob (charts/lfx-crowdfunding-backend/templates/cronjob-b2b-org-backfill.yaml).
//
// Ledger rows are filled separately: --export writes domain,sfid for every
// matched domain, the input for ledger-service's
// scripts/backfill-organization-claim --sfid-csv.
//
// Usage:
//
//	go run ./cmd/b2b-org-backfill [--dry-run] [--export org_sfids.csv]
//
// Flags:
//
//	--dry-run  Look up domains and report matches without writing anything.
//	--export   After the run, write domain,sfid for every matched domain to this path.
//	           Can't be combined with --dry-run, which saves no matches to export.
//
// Required env vars: DATABASE_URL, FGA_NATS_URL, STRIPE_SECRET_KEY.
//
// Exits 0 on success (including per-domain lookup or Stripe failures, which are
// logged and retried on the next run), non-zero if it cannot start or a DB write fails.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stripe/stripe-go/v85"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/clients"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/db"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	dryRun := flag.Bool("dry-run", false, "Look up domains and report matches without writing")
	export := flag.String("export", "", "Write domain,sfid for every matched domain to this path")
	flag.Parse()
	if *dryRun && *export != "" {
		logger.Error("--export can't be combined with --dry-run: the export reads saved matches and a dry run saves none")
		os.Exit(2)
	}

	if err := run(logger, *dryRun, *export); err != nil {
		logger.Error("b2b-org-backfill failed", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger, dryRun bool, exportPath string) error {
	ctx := context.Background()
	start := time.Now()

	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	pool, err := db.NewPool(ctx, db.PoolConfig{
		DSN:             cfg.DatabaseURL,
		MaxConns:        2,
		MinConns:        1,
		ConnMaxLifetime: 5 * time.Minute,
	})
	if err != nil {
		return fmt.Errorf("database pool: %w", err)
	}
	defer pool.Close()

	conn, err := nats.Connect(cfg.NATSURL, nats.Timeout(10*time.Second))
	if err != nil {
		return fmt.Errorf("nats connect: %w", err)
	}
	defer conn.Close()

	st := &pgStore{pool: pool}
	b := &backfiller{
		store:  st,
		lookup: clients.NewMemberClient(conn),
		stripe: stripePatcher{api: stripe.NewClient(cfg.StripeSecretKey)},
		dryRun: dryRun,
		logger: logger,
	}

	logger.Info("b2b-org-backfill starting", "dry_run", dryRun)
	res, err := b.Run(ctx)
	if err != nil {
		return fmt.Errorf("backfill run: %w", err)
	}
	logger.Info("b2b-org-backfill complete",
		"dry_run", dryRun,
		"duration", time.Since(start).String(),
		"domains", res.domains,
		"matched", res.matched,
		"no_match", res.noMatch,
		"lookup_failed", res.lookupFailed,
		"donations_filled", res.donations,
		"subscriptions_filled", res.subscriptions,
		"stripe_patched", res.stripePatched,
		"stripe_skipped", res.stripeSkipped,
		"stripe_failed", res.stripeFailed,
	)

	if exportPath == "" {
		return nil
	}
	f, err := os.Create(exportPath)
	if err != nil {
		return fmt.Errorf("create export: %w", err)
	}
	defer f.Close() //nolint:errcheck // closed and checked below; this covers the error paths
	n, conflicts, err := st.ExportMatches(ctx, f)
	if err != nil {
		return fmt.Errorf("export matches: %w", err)
	}
	logger.Info("exported matched domains", "path", exportPath, "domains", n, "conflicts", conflicts)
	return f.Close()
}

type config struct {
	DatabaseURL     string
	NATSURL         string
	StripeSecretKey string
}

func loadConfig() (*config, error) {
	cfg := &config{
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		NATSURL:         os.Getenv("FGA_NATS_URL"),
		StripeSecretKey: os.Getenv("STRIPE_SECRET_KEY"),
	}
	for name, v := range map[string]string{"DATABASE_URL": cfg.DatabaseURL, "FGA_NATS_URL": cfg.NATSURL, "STRIPE_SECRET_KEY": cfg.StripeSecretKey} {
		if v == "" {
			return nil, fmt.Errorf("%s is required", name)
		}
	}
	return cfg, nil
}
