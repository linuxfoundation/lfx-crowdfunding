// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Wire shapes of the v2 query service's `GET /query/resources`, as read through the gateway.

export interface QueryResource<T> {
  type: string;
  // `<type>:<uid>`, e.g. `b2b_org_settings:0014100000AcmeAAAA` or `project:<uuid>`.
  id: string;
  data: T;
}

export interface QueryResourcesResponse<T> {
  resources: QueryResource<T>[];
  page_token?: string;
}

// Display fields shared by `b2b_org` and `project` documents.
export interface EntityDoc {
  name?: string;
  slug?: string;
  logo_url?: string | null;
}

export interface OrgSettingsMember {
  username?: string | null;
  role?: 'writer' | 'auditor';
  invite_status?: 'pending' | 'accepted' | 'revoked' | 'expired';
}

// `b2b_org_settings` — current indexer shape is `members[]`; `writers[]`/`auditors[]` is the
// legacy pre-flatten shape still present on older docs.
export interface OrgSettingsDoc {
  members?: OrgSettingsMember[];
  writers?: OrgSettingsMember[];
  auditors?: OrgSettingsMember[];
}
