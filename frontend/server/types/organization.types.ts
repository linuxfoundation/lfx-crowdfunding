// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

export interface OrganizationResponse {
  id: string;
  owner_id: string;
  name: string;
  avatar_url?: string;
  status?: string;
  created_on: string;
  updated_on: string;
}
