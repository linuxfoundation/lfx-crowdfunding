<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div
    v-if="modeOptions.length > 1"
    class="border border-neutral-200 rounded-xl flex items-stretch overflow-hidden"
  >
    <div
      v-for="(option, index) in modeOptions"
      :key="option.value"
      role="button"
      tabindex="0"
      class="flex-1 flex gap-3 p-5 cursor-pointer"
      :class="[
        modelValue === option.value ? 'bg-accent-50' : 'bg-white hover:bg-neutral-50',
        index < modeOptions.length - 1 ? 'border-r border-neutral-200' : '',
      ]"
      @click="emit('update:modelValue', option.value)"
      @keydown.enter="emit('update:modelValue', option.value)"
      @keydown.space.prevent="emit('update:modelValue', option.value)"
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
import { DONATION_MODE_OPTIONS } from '../../../config/donation-options.config';
import type { DonationOptionsMode } from '~/types/fundraise.types';
import LfxRadio from '~/components/uikit/radio/radio.vue';

defineProps<{
  modelValue: DonationOptionsMode;
}>();

const modeOptions = DONATION_MODE_OPTIONS;

const emit = defineEmits<{
  (e: 'update:modelValue', value: DonationOptionsMode): void;
}>();
</script>

<script lang="ts">
export default {
  name: 'FundraiseDonationModeSection',
};
</script>
