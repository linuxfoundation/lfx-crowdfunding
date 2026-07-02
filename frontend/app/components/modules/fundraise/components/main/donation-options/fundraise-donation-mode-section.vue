<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="border border-neutral-200 rounded-xl flex items-stretch overflow-hidden">
    <div
      v-for="(option, index) in modeOptions"
      :key="option.value"
      class="flex-1 flex gap-3 p-5 cursor-pointer"
      :class="[
        modelValue === option.value ? 'bg-accent-50' : 'bg-white hover:bg-neutral-50',
        index < modeOptions.length - 1 ? 'border-r border-neutral-200' : '',
      ]"
      @click="emit('update:modelValue', option.value)"
    >
      <lfx-radio
        :model-value="modelValue"
        :value="option.value"
        @update:model-value="emit('update:modelValue', $event as DonationOptionsMode)"
      />

      <div class="flex flex-col gap-2">
        <span class="text-sm font-semibold text-neutral-900">{{ option.label }}</span>
        <p class="text-xs text-neutral-600 leading-4">{{ option.description }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { DONATION_MODE_OPTIONS } from '../../../config/donation-options.config';
import type { DonationOptionsMode } from '~/types/fundraise.types';
import LfxRadio from '~/components/uikit/radio/radio.vue';
import { isSponsorshipTiersEnabled } from '~/utils/feature-flags';

defineProps<{
  modelValue: DonationOptionsMode;
}>();

const modeOptions = computed(() =>
  isSponsorshipTiersEnabled() ? DONATION_MODE_OPTIONS : DONATION_MODE_OPTIONS.filter((o) => o.value !== 'tiers'),
);

const emit = defineEmits<{
  (e: 'update:modelValue', value: DonationOptionsMode): void;
}>();
</script>

<script lang="ts">
export default {
  name: 'FundraiseDonationModeSection',
};
</script>
