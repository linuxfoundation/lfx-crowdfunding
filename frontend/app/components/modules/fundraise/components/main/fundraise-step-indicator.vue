<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="flex gap-4">
    <div
      v-for="(label, i) in steps"
      :key="i"
      class="flex-1 flex flex-col gap-2"
    >
      <div
        class="h-1 rounded-full"
        :class="i <= currentStep ? 'bg-accent-500' : 'bg-neutral-200'"
      />
      <span
        class="text-xs leading-4"
        :class="i === currentStep ? 'font-semibold text-neutral-900' : 'text-neutral-500'"
      >
        {{ label }}
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  steps: string[];
  currentStep: number;
}>();
</script>

<script lang="ts">
export default {
  name: 'FundraiseStepIndicator',
};
</script>
