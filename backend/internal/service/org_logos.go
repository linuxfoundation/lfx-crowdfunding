// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"log/slog"
)

// orgLogoReader is the read side of domain.OrgLogoRepository.
type orgLogoReader interface {
	LogosByDomains(ctx context.Context, domains []string) (map[string]string, error)
}

// orgLogos returns the b2b_org logo for each org domain that has one (#472).
// Donor lists show generated initials for the rest, so a nil reader or a
// failed lookup returns no logos instead of failing the list.
func orgLogos(ctx context.Context, r orgLogoReader, domains []string) map[string]string {
	if r == nil || len(domains) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(domains))
	unique := make([]string, 0, len(domains))
	for _, d := range domains {
		if _, ok := seen[d]; d != "" && !ok {
			seen[d] = struct{}{}
			unique = append(unique, d)
		}
	}
	if len(unique) == 0 {
		return nil
	}
	logos, err := r.LogosByDomains(ctx, unique)
	if err != nil {
		slog.WarnContext(ctx, "org logo lookup failed, using generated avatars", "error", err)
		return nil
	}
	return logos
}
