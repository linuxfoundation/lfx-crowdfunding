// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// `GET /crowdfunding/me/org-logos`: the b2b_org logos CF has stored, keyed by the
// requested domain. Domains without a logo are absent (#472).
export interface OrgLogosWire {
  logos: Record<string, string>;
}
