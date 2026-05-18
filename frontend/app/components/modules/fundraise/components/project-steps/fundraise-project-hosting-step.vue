<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="flex flex-col gap-6">
    <div class="flex flex-col gap-1">
      <h2 class="text-base font-semibold text-neutral-900">How is your project hosted?</h2>
      <p class="text-sm text-neutral-600 leading-5">
        Connect via GitHub for the best experience, or provide any Git repository URL.
      </p>
    </div>

    <div class="border border-neutral-200 rounded-xl overflow-hidden">
      <button
        v-for="(option, index) in HOSTING_OPTIONS"
        :key="option.value"
        type="button"
        class="w-full flex items-start gap-4 p-6 text-left transition-colors"
        :class="[
          modelValue === option.value ? 'bg-accent-50' : 'bg-white hover:bg-neutral-50',
          index < HOSTING_OPTIONS.length - 1 ? 'border-b border-neutral-200' : '',
        ]"
        @click="$emit('update:modelValue', option.value)"
      >
        <!-- Icon -->
        <div
          class="shrink-0 size-10 rounded-full bg-white border border-neutral-200 shadow-sm flex items-center justify-center"
        >
          <lfx-icon
            :name="option.icon"
            :type="option.iconType"
            :size="20"
            class="text-neutral-900"
          />
        </div>

        <!-- Label + badge + description -->
        <div class="flex-1 min-w-0 flex flex-col gap-1 justify-center">
          <div class="flex items-center gap-2">
            <span class="text-sm font-semibold text-neutral-900 leading-5">{{ option.label }}</span>
            <lfx-tag
              v-if="option.recommended"
              variation="positive"
              size="small"
            >
              Recommended
            </lfx-tag>
          </div>
          <p class="text-xs text-neutral-600 leading-4">{{ option.description }}</p>
        </div>

        <!-- Radio indicator -->
        <div
          class="shrink-0 size-4 rounded-full border flex items-center justify-center mt-0.5"
          :class="modelValue === option.value ? 'bg-accent-500 border-accent-500' : 'bg-white border-neutral-300'"
        >
          <div
            v-if="modelValue === option.value"
            class="size-1.5 rounded-full bg-white"
          />
        </div>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { ProjectHostingType } from '~/types/fundraise.types';
import LfxIcon from '~/components/uikit/icon/icon.vue';
import LfxTag from '~/components/uikit/tag/tag.vue';
import type { IconType } from '~/components/uikit/icon/types/icon.types';

interface HostingOption {
  value: ProjectHostingType;
  label: string;
  description: string;
  icon: string;
  iconType: IconType;
  recommended?: boolean;
}

const HOSTING_OPTIONS: HostingOption[] = [
  {
    value: 'github',
    label: 'GitHub',
    description: 'Connect via OAuth to select a repository. Enables stars, contributors, and health metrics.',
    icon: 'github',
    iconType: 'brands',
    recommended: true,
  },
  {
    value: 'git_url',
    label: 'Git URL',
    description: 'Provide any public Git repository URL. Works with GitLab, Gitea, Bitbucket, or self-hosted repos.',
    icon: 'link',
    iconType: 'light',
  },
];

defineProps<{
  modelValue: ProjectHostingType | null;
}>();

defineEmits<{
  (e: 'update:modelValue', value: ProjectHostingType): void;
}>();
</script>

<script lang="ts">
export default {
  name: 'FundraiseProjectHostingStep',
};
</script>
