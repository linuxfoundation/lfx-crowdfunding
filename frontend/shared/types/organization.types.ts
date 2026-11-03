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
