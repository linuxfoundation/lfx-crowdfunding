// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package migrations embeds the SQL migration files so cmd/migrate can apply
// them from a single binary with no separate file mount.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
