<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="flex flex-col gap-6">
    <div class="flex flex-col gap-1">
      <h2 class="text-base font-semibold text-neutral-900">Donation options</h2>
      <p class="text-sm text-neutral-600 leading-5">
        {{
          isSponsorshipTiersEnabled()
            ? 'Set up donation tiers to give contributors a clear choice of giving levels, or skip this and collect open-amount donations only.'
            : 'Your fundraiser will collect open-amount donations.'
        }}
      </p>
    </div>

    <fundraise-donation-mode-section
      :model-value="modelValue.mode"
      @update:model-value="emit('update:modelValue', { ...modelValue, mode: $event })"
    />

    <fundraise-donation-tiers-section
      v-if="modelValue.mode === 'tiers' && isSponsorshipTiersEnabled()"
      :model-value="modelValue.tiers"
      @update:model-value="emit('update:modelValue', { ...modelValue, tiers: $event })"
    />
  </div>
</template>

<script setup lang="ts">
import FundraiseDonationModeSection from './donation-options/fundraise-donation-mode-section.vue';
import FundraiseDonationTiersSection from './donation-options/fundraise-donation-tiers-section.vue';
import type { DonationOptionsData } from '~/types/fundraise.types';
import { isSponsorshipTiersEnabled } from '~/utils/feature-flags';

defineProps<{
  modelValue: DonationOptionsData;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: DonationOptionsData): void;
}>();
</script>

<script lang="ts">
export default {
  name: 'FundraiseDonationOptionsStep',
};
</script>
