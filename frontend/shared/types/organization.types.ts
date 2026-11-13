// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

/**
 * The organization a donor gives on behalf of, as stored on the donation (#365):
 * a name and domain. Also the shape of an org search (Clearbit) result.
 */
export interface OrgClaim {
  name: string;
  domain: string;
}

/**
 * An org search result: the claim plus the b2b_org logo CF has stored for that
 * domain, when an org donation for it was matched before (#472). Only name and
 * domain are sent when donating.
 */
export interface OrgSuggestion extends OrgClaim {
  logoUrl?: string;
}
