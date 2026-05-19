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
      <div
        v-if="isInitializing"
        class="flex items-center justify-center py-10 text-sm text-neutral-500"
      >
        Loading…
      </div>

      <template v-else-if="!isConnected">
        <github-connect-prompt @connect="connectWithSession" />
      </template>

      <template v-else-if="isConnected">
        <github-connected-header
          :login="user!.login"
          @disconnect="disconnect"
        />

        <div class="border-t border-neutral-200" />

        <github-repo-list
          :repos="repos"
          :is-loading="isLoadingRepos"
          :model-value="modelValue"
          @update:model-value="$emit('update:modelValue', $event)"
        />
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { useRoute, useRouter } from 'nuxt/app';
import GithubConnectPrompt from './github-connect-prompt.vue';
import GithubConnectedHeader from './github-connected-header.vue';
import GithubRepoList from './github-repo-list.vue';
import { useGithubAuth } from '~/composables/useGithubAuth';

defineProps<{
  modelValue: string | null;
}>();

defineEmits<{
  (e: 'update:modelValue', value: string): void;
}>();

const route = useRoute();
const router = useRouter();

const { isConnected, user, repos, isLoadingRepos, fetchUser, fetchRepos, connect, disconnect } = useGithubAuth();

const connectWithSession = () => {
  connect({ initiativeType: 'project', step: 1, subStep: 1, hostingType: 'github' });
};

const isInitializing = ref(true);

onMounted(async () => {
  try {
    if (route.query.github_connected) {
      await router.replace({ query: { ...route.query, github_connected: undefined } });
    }

    await fetchUser();

    if (isConnected.value) {
      await fetchRepos();
    }
  } finally {
    isInitializing.value = false;
  }
});
</script>

<script lang="ts">
export default {
  name: 'FundraiseProjectGithubStep',
};
</script>
