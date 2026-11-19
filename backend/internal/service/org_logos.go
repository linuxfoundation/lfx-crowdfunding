// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"log/slog"
)

// orgLogoReader is the read side of domain.OrgLogoRepository.
type orgLogoReader interface {
	LogosByOrgKeys(ctx context.Context, keys []string) (map[string]string, error)
}

// orgLogos returns the logo for each Ledger org key that has one: the matched
// b2b_org's logo for a domain (#472), the legacy CF org's uploaded logo for a
// legacy org UUID (lfx-self-serve-ops#1647).
// Donor lists show generated initials for the rest, so a nil reader or a
// failed lookup returns no logos instead of failing the list.
func orgLogos(ctx context.Context, r orgLogoReader, keys []string) map[string]string {
	if r == nil || len(keys) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(keys))
	unique := make([]string, 0, len(keys))
	for _, k := range keys {
		if _, ok := seen[k]; k != "" && !ok {
			seen[k] = struct{}{}
			unique = append(unique, k)
		}
	}
	if len(unique) == 0 {
		return nil
	}
	logos, err := r.LogosByOrgKeys(ctx, unique)
	if err != nil {
		slog.WarnContext(ctx, "org logo lookup failed, using generated avatars", "error", err)
		return nil
	}
	return logos
}
