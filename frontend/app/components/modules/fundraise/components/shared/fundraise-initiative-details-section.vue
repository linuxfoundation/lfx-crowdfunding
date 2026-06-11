<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="border border-neutral-200 rounded-xl p-6">
    <div class="flex flex-col gap-5">
      <h2 class="text-base font-semibold text-neutral-900">{{ title }}</h2>

      <!-- Name -->
      <div class="flex flex-col gap-3">
        <div class="flex items-center justify-between">
          <label class="text-xs font-medium text-neutral-900">
            {{ nameLabel }} <span class="text-negative-500">*</span>
          </label>
          <span class="text-xs text-neutral-500">{{ modelValue.name.length }}/100</span>
        </div>
        <lfx-input
          :model-value="modelValue.name"
          placeholder="My project"
          @update:model-value="update('name', $event as string)"
        />
      </div>

      <!-- Elevator Pitch -->
      <div class="flex flex-col gap-3">
        <div class="flex items-center justify-between">
          <label class="text-xs font-medium text-neutral-900">
            Elevator Pitch <span class="text-negative-500">*</span>
          </label>
          <span
            class="text-xs"
            :class="modelValue.elevatorPitch.length > 5000 ? 'text-negative-500' : 'text-neutral-500'"
          >
            {{ modelValue.elevatorPitch.length }}/5000
          </span>
        </div>
        <lfx-textarea
          :model-value="modelValue.elevatorPitch"
          placeholder="Briefly introduce your project..."
          class="h-[72px]"
          :maxlength="5000"
          @update:model-value="update('elevatorPitch', $event as string)"
        />
      </div>

      <!-- Topic / Category -->
      <div class="flex flex-col gap-3">
        <label class="text-xs font-medium text-neutral-900">
          Topic / Category <span class="text-negative-500">*</span>
        </label>
        <fundraise-project-topic-select
          :model-value="modelValue.topics"
          @update:model-value="update('topics', $event)"
        />
      </div>

      <!-- Repository URL (optional) -->
      <div
        v-if="showRepositoryUrl"
        class="flex flex-col gap-3"
      >
        <div class="flex flex-col gap-1">
          <label class="text-xs font-medium text-neutral-900">Repository URL</label>
          <p class="text-xs text-neutral-600 leading-4">
            This URL will be used to display repository statistics on your LFX Crowdfunding page.
          </p>
        </div>
        <lfx-input
          :model-value="modelValue.repositoryUrl ?? ''"
          placeholder="https://github.com/org/repo"
          @update:model-value="update('repositoryUrl', $event as string)"
        />
      </div>

      <!-- Website URL -->
      <div class="flex flex-col gap-3">
        <label class="text-xs font-medium text-neutral-900">Website URL</label>
        <lfx-input
          :model-value="modelValue.websiteUrl"
          placeholder="https://example.org"
          @update:model-value="update('websiteUrl', $event as string)"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import FundraiseProjectTopicSelect from '../project-steps/details/fundraise-project-topic-select.vue';
import type { InitiativeDetailsData } from '~/types/fundraise.types';
import LfxInput from '~/components/uikit/input/input.vue';
import LfxTextarea from '~/components/uikit/textarea/textarea.vue';

const props = defineProps<{
  modelValue: InitiativeDetailsData;
  title: string;
  nameLabel: string;
  showRepositoryUrl?: boolean;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: InitiativeDetailsData): void;
}>();

const update = <K extends keyof InitiativeDetailsData>(key: K, value: InitiativeDetailsData[K]) => {
  emit('update:modelValue', { ...props.modelValue, [key]: value });
};
</script>

<script lang="ts">
export default {
  name: 'FundraiseInitiativeDetailsSection',
};
</script>
