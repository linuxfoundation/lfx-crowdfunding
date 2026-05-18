<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="border border-neutral-200 rounded-xl p-6">
    <div class="flex flex-col gap-5">
      <h2 class="text-base font-semibold text-neutral-900">Project Details</h2>

      <!-- Project name -->
      <div class="flex flex-col gap-3">
        <div class="flex items-center justify-between">
          <label class="text-xs font-medium text-neutral-900">
            Project name <span class="text-negative-500">*</span>
          </label>
          <span class="text-xs text-neutral-500">{{ modelValue.projectName.length }}/100</span>
        </div>
        <lfx-input
          :model-value="modelValue.projectName"
          placeholder="My project"
          @update:model-value="update('projectName', $event as string)"
        />
      </div>

      <!-- Elevator Pitch -->
      <div class="flex flex-col gap-3">
        <div class="flex items-center justify-between">
          <label class="text-xs font-medium text-neutral-900">
            Elevator Pitch <span class="text-negative-500">*</span>
          </label>
          <span class="text-xs text-neutral-500">{{ modelValue.elevatorPitch.length }}/500</span>
        </div>
        <lfx-textarea
          :model-value="modelValue.elevatorPitch"
          placeholder="Briefly introduce your project..."
          class="h-[72px]"
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
import FundraiseProjectTopicSelect from './fundraise-project-topic-select.vue';
import type { ProjectDetailsData } from '~/types/fundraise.types';
import LfxInput from '~/components/uikit/input/input.vue';
import LfxTextarea from '~/components/uikit/textarea/textarea.vue';

const props = defineProps<{
  modelValue: ProjectDetailsData;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: ProjectDetailsData): void;
}>();

const update = <K extends keyof ProjectDetailsData>(key: K, value: ProjectDetailsData[K]) => {
  emit('update:modelValue', { ...props.modelValue, [key]: value });
};
</script>

<script lang="ts">
export default {
  name: 'FundraiseProjectDetailsSection',
};
</script>
