// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { GitHubRepo } from '~/types/fundraise.types';

interface GitHubApiRepo {
  id: number;
  name: string;
  full_name: string;
  description: string | null;
  stargazers_count: number;
}

export default defineEventHandler(async (event) => {
  const token = getCookie(event, 'github_oauth_token');

  if (!token) {
    throw createError({ statusCode: 401, statusMessage: 'GitHub account not connected' });
  }

  const apiRepos = await $fetch<GitHubApiRepo[]>('https://api.github.com/user/repos', {
    headers: {
      Authorization: `Bearer ${token}`,
      Accept: 'application/vnd.github+json',
    },
    query: {
      sort: 'updated',
      per_page: 100,
      visibility: 'public',
    },
  });

  const repos: GitHubRepo[] = apiRepos.map((r) => ({
    id: r.id,
    name: r.name,
    fullName: r.full_name,
    description: r.description || '',
    stars: r.stargazers_count,
  }));

  return repos;
});
