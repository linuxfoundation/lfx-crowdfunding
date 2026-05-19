<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="flex flex-col gap-4 p-6">
    <div class="flex flex-col gap-1">
      <p class="text-sm font-semibold text-neutral-900">
        Select a repository
        <span class="text-negative-500 ml-0.5">*</span>
      </p>
      <p class="text-xs text-neutral-500 leading-4">Choose the GitHub repository this initiative will be linked to.</p>
    </div>

    <lfx-input
      v-model="search"
      placeholder="Search repositories..."
    >
      <template #prefix>
        <lfx-icon
          name="magnifying-glass"
          type="light"
          :size="14"
          class="text-neutral-400"
        />
      </template>
    </lfx-input>

    <div
      v-if="isLoading"
      class="flex items-center justify-center py-10 text-sm text-neutral-500"
    >
      Loading repositories…
    </div>

    <div
      v-else
      class="flex flex-col -mx-6"
    >
      <button
        v-for="(repo, index) in filteredRepos"
        :key="repo.id"
        type="button"
        class="flex items-center gap-3 px-6 py-4 text-left transition-colors"
        :class="[
          modelValue === repo.fullName ? 'bg-accent-50' : 'hover:bg-neutral-50',
          index < filteredRepos.length - 1 ? 'border-b border-neutral-200' : '',
        ]"
        @click="$emit('update:modelValue', repo.fullName)"
      >
        <div
          class="shrink-0 size-4 rounded-full border flex items-center justify-center"
          :class="modelValue === repo.fullName ? 'bg-accent-500 border-accent-500' : 'bg-white border-neutral-300'"
        >
          <div
            v-if="modelValue === repo.fullName"
            class="size-1.5 rounded-full bg-white"
          />
        </div>

        <div class="flex-1 min-w-0">
          <p class="text-sm font-semibold text-neutral-900 leading-5">{{ repo.name }}</p>
          <p class="text-xs text-neutral-500 leading-4 truncate">{{ repo.description }}</p>
        </div>

        <div class="shrink-0 flex items-center gap-1 text-xs text-neutral-500">
          <lfx-icon
            name="star"
            type="light"
            :size="12"
          />
          <span>{{ formatStars(repo.stars) }}</span>
        </div>
      </button>

      <p
        v-if="filteredRepos.length === 0"
        class="px-6 py-4 text-xs text-neutral-500"
      >
        No repositories match your search.
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import type { GitHubRepo } from '~/types/fundraise.types';
import LfxIcon from '~/components/uikit/icon/icon.vue';
import LfxInput from '~/components/uikit/input/input.vue';

const props = defineProps<{
  repos: GitHubRepo[];
  modelValue: string | null;
  isLoading?: boolean;
}>();

defineEmits<{
  (e: 'update:modelValue', value: string): void;
}>();

const search = ref('');

const filteredRepos = computed(() => {
  const q = search.value.trim().toLowerCase();
  if (!q) return props.repos;
  return props.repos.filter((r) => r.name.toLowerCase().includes(q) || r.description.toLowerCase().includes(q));
});

const formatStars = (count: number): string => {
  if (count >= 1000) return `${Math.round(count / 1000)}K`;
  return String(count);
};
</script>

<script lang="ts">
export default {
  name: 'GithubRepoList',
};
</script>
