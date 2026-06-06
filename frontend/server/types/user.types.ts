// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

export interface UserResponse {
  id: string;
  username: string;
  legacy_user_id?: string;
  email?: string;
  given_name?: string;
  family_name?: string;
  name?: string;
  avatar_url?: string;
  created_on: string;
  updated_on: string;
}
