// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { defineEventHandler } from 'h3';
import { useBackendFetch } from '../utils/backend-fetch';
import type { UserResponse } from '../types/user.types';

// Authentication is enforced by server/middleware/require-auth.ts which guards
// PATCH /api/me and rejects requests missing auth_oidc_token.
export default defineEventHandler((event): Promise<UserResponse> => {
  return useBackendFetch<UserResponse>(event, '/v1/me', {
    method: 'PATCH',
  });
});
