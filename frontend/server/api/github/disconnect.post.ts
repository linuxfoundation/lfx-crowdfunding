// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

export default defineEventHandler((event) => {
  deleteCookie(event, 'github_oauth_token');
  return { success: true };
});
