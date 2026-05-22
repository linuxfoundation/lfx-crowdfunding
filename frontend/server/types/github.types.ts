// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

export interface GitHubApiRepo {
  id: number;
  name: string;
  full_name: string;
  description: string | null;
  stargazers_count: number;
}
