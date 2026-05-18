<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="border border-neutral-200 rounded-xl p-6">
    <div class="flex flex-col gap-5">
      <h2 class="text-base font-semibold text-neutral-900">Date &amp; Location</h2>

      <!-- Start / End date -->
      <div class="grid grid-cols-2 gap-6">
        <div class="flex flex-col gap-2">
          <label class="text-xs font-medium text-neutral-900">
            Start date <span class="text-negative-500">*</span>
          </label>
          <lfx-input
            :model-value="modelValue.startDate"
            placeholder="MM/DD/YYYY"
            @update:model-value="update('startDate', $event as string)"
          />
        </div>
        <div class="flex flex-col gap-2">
          <label class="text-xs font-medium text-neutral-900">
            End date <span class="text-negative-500">*</span>
          </label>
          <lfx-input
            :model-value="modelValue.endDate"
            placeholder="MM/DD/YYYY"
            @update:model-value="update('endDate', $event as string)"
          />
        </div>
      </div>

      <!-- City / Country -->
      <div class="grid grid-cols-2 gap-6">
        <div class="flex flex-col gap-2">
          <label class="text-xs font-medium text-neutral-900">City</label>
          <lfx-input
            :model-value="modelValue.city"
            placeholder=""
            @update:model-value="update('city', $event as string)"
          />
        </div>
        <div class="flex flex-col gap-2">
          <label class="text-xs font-medium text-neutral-900">Country</label>
          <lfx-input
            :model-value="modelValue.country"
            placeholder=""
            @update:model-value="update('country', $event as string)"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { EventFormData } from '~/types/fundraise.types';
import LfxInput from '~/components/uikit/input/input.vue';

const props = defineProps<{
  modelValue: EventFormData;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: EventFormData): void;
}>();

const update = <K extends keyof EventFormData>(key: K, value: EventFormData[K]) => {
  emit('update:modelValue', { ...props.modelValue, [key]: value });
};
</script>

<script lang="ts">
export default {
  name: 'FundraiseEventDateLocationSection',
};
</script>
