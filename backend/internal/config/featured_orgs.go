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
)

//go:embed featured_orgs.json
var featuredOrgsJSON []byte

// featuredOrg is one entry in featured_orgs.json. Name is a human-readable
// comment for maintainers; the database remains authoritative for the
// rendered organization name.
type featuredOrg struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// featuredOrgIDs is the parsed, ordered list of organization IDs curated for
// the "/for-companies" investing-companies section. Order is preserved from
// featured_orgs.json and used as the display order in prod.
var featuredOrgIDs = mustParseFeaturedOrgIDs(featuredOrgsJSON)

func mustParseFeaturedOrgIDs(raw []byte) []string {
	var orgs []featuredOrg
	if err := json.Unmarshal(raw, &orgs); err != nil {
		panic(fmt.Sprintf("config: failed to parse featured_orgs.json: %v", err))
	}
	ids := make([]string, len(orgs))
	for i, o := range orgs {
		if o.ID == "" {
			panic("config: featured_orgs.json contains an entry with an empty id")
		}
		ids[i] = o.ID
	}
	return ids
}

// FeaturedOrgIDs returns the curated, ordered list of organization IDs for the
// investing-companies section. These IDs come from the production database
// (see featured_orgs.json) and generally won't resolve to rows in local, dev,
// or staging databases.
func FeaturedOrgIDs() []string {
	return featuredOrgIDs
}
