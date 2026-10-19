// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

export interface ProtectedRoute {
  match: (path: string) => boolean;
  methods?: string[];
}

// Display-only profile claims stored (encrypted) in the auth_user_profile cookie.
export interface UserProfile {
  sub: string;
  name?: string;
  email?: string;
  picture?: string;
  email_verified?: boolean;
  updated_at?: string;
  username?: string;
  intercomJwt?: string;
}
