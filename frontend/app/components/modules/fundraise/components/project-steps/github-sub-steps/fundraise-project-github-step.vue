<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="flex flex-col gap-6">
    <div class="flex flex-col gap-1">
      <h2 class="text-base font-semibold text-neutral-900">Connect GitHub</h2>
      <p class="text-sm text-neutral-600 leading-5">
        Connect your GitHub account so we can link your initiative to a repository.
      </p>
    </div>

    <div class="border border-neutral-200 rounded-xl overflow-hidden">
      <github-connect-prompt
        v-if="!isConnected"
        @connect="isConnected = true"
      />

      <template v-else>
        <github-connected-header :login="MOCK_USER.login" />

        <div class="border-t border-neutral-200" />

        <github-repo-list
          :repos="MOCK_REPOS"
          :model-value="modelValue"
          @update:model-value="$emit('update:modelValue', $event)"
        />
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import GithubConnectPrompt from './github-connect-prompt.vue';
import GithubConnectedHeader from './github-connected-header.vue';
import GithubRepoList from './github-repo-list.vue';
import type { GitHubRepo } from '~/types/fundraise.types';

const MOCK_USER = { login: 'nunoeufrasio', name: 'Nuno Eufrasio' };

const MOCK_REPOS: GitHubRepo[] = [
  {
    id: 1,
    fullName: 'kubernetes/kubernetes',
    name: 'kubernetes/kubernetes',
    description: 'Production-Grade Container Scheduling and Management',
    stars: 110000,
  },
  { id: 2, fullName: 'torvalds/linux', name: 'torvalds/linux', description: 'Linux kernel source tree', stars: 185000 },
  { id: 3, fullName: 'golang/go', name: 'golang/go', description: 'The Go programming language', stars: 123000 },
  {
    id: 4,
    fullName: 'python/cpython',
    name: 'python/cpython',
    description: 'The Python programming language',
    stars: 62000,
  },
  {
    id: 5,
    fullName: 'nodejs/node',
    name: 'nodejs/node',
    description: "Node.js JavaScript runtime built on Chrome's V8 engine",
    stars: 107000,
  },
  {
    id: 6,
    fullName: 'rust-lang/rust',
    name: 'rust-lang/rust',
    description: 'Empowering everyone to build reliable and efficient software',
    stars: 97000,
  },
];

defineProps<{
  modelValue: string | null;
}>();

defineEmits<{
  (e: 'update:modelValue', value: string): void;
}>();

const isConnected = ref(false);
</script>

<script lang="ts">
export default {
  name: 'FundraiseProjectGithubStep',
};
</script>
