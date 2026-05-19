// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { ref, computed } from 'vue';
import { useRoute } from 'nuxt/app';
import type { GitHubRepo } from '~/types/fundraise.types';

export const GITHUB_FUNDRAISE_SESSION_KEY = 'github_fundraise_session';

export interface GitHubFundraiseSession {
  initiativeType: string;
  step: number;
  subStep: number;
  hostingType: string;
}

interface GitHubUser {
  login: string;
  name: string | null;
}

const githubUser = ref<GitHubUser | null>(null);
const githubRepos = ref<GitHubRepo[]>([]);
const isLoadingUser = ref(false);
const isLoadingRepos = ref(false);

export const useGithubAuth = () => {
  const isConnected = computed(() => githubUser.value !== null);

  const fetchUser = async () => {
    isLoadingUser.value = true;
    try {
      const user = await $fetch<GitHubUser | null>('/api/github/user');
      githubUser.value = user ?? null;
    } catch {
      githubUser.value = null;
    } finally {
      isLoadingUser.value = false;
    }
  };

  const fetchRepos = async () => {
    if (!isConnected.value) return;
    isLoadingRepos.value = true;
    try {
      githubRepos.value = await $fetch<GitHubRepo[]>('/api/github/repos');
    } catch {
      githubRepos.value = [];
    } finally {
      isLoadingRepos.value = false;
    }
  };

  const connect = (session?: GitHubFundraiseSession) => {
    if (session) {
      sessionStorage.setItem(GITHUB_FUNDRAISE_SESSION_KEY, JSON.stringify(session));
    }
    const route = useRoute();
    const redirectTo = encodeURIComponent(route.fullPath);
    window.location.href = `/api/github/authorize?redirectTo=${redirectTo}`;
  };

  const disconnect = async () => {
    await $fetch('/api/github/disconnect', { method: 'POST' });
    githubUser.value = null;
    githubRepos.value = [];
  };

  return {
    isConnected,
    user: githubUser,
    repos: githubRepos,
    isLoadingUser,
    isLoadingRepos,
    fetchUser,
    fetchRepos,
    connect,
    disconnect,
  };
};
