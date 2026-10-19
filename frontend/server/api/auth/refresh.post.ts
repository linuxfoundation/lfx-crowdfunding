// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

export default defineEventHandler(async (event) => {
  if (!(await refreshSession(event))) {
    throw createError({ statusCode: 401, statusMessage: 'Token refresh failed' });
  }
  return { success: true };
});
