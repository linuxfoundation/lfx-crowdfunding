// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package config holds embedded static configuration for the initiatives API,
// as opposed to runtime environment-variable configuration
// (see cmd/initiatives-api/config.go).
package config

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
)

//go:embed featured_orgs.json
var featuredOrgsJSON []byte

// FeaturedOrg is one entry in featured_orgs.json, keyed by the normalized
// domain Ledger groups org donations on (#365). LegacyID is the org's prod CF
// UUID, which Ledger still buckets on for donations not yet backfilled with a
// domain (#408).
type FeaturedOrg struct {
	Domain   string `json:"domain"`
	Name     string `json:"name"`
	LogoURL  string `json:"logo_url"`
	LegacyID string `json:"legacy_id"`
}

// featuredOrgs is the parsed, ordered list of organizations curated for the
// "/for-companies" investing-companies section. Order is preserved from
// featured_orgs.json and used as the display order.
var featuredOrgs = mustParseFeaturedOrgs(featuredOrgsJSON)

func mustParseFeaturedOrgs(raw []byte) []FeaturedOrg {
	var orgs []FeaturedOrg
	if err := json.Unmarshal(raw, &orgs); err != nil {
		panic(fmt.Sprintf("config: failed to parse featured_orgs.json: %v", err))
	}
	for _, o := range orgs {
		if d, err := domain.NormalizeDomain(o.Domain); err != nil || d != o.Domain {
			panic(fmt.Sprintf("config: featured_orgs.json domain %q is not a normalized domain", o.Domain))
		}
	}
	return orgs
}

// FeaturedOrgs returns the curated, ordered list of organizations for the
// investing-companies section.
func FeaturedOrgs() []FeaturedOrg {
	return featuredOrgs
}
