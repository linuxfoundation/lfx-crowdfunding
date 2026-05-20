// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

export default defineEventHandler(async (event) => {
  const token = getCookie(event, 'github_oauth_token');

  if (!token) {
    return null;
  }

  const user = await $fetch<{ login: string; name: string | null }>('https://api.github.com/user', {
    headers: {
      Authorization: `Bearer ${token}`,
      Accept: 'application/vnd.github+json',
    },
  });

  return { login: user.login, name: user.name };
});
