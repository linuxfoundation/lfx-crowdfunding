<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="border border-neutral-200 rounded-xl p-6">
    <div class="flex flex-col gap-5">
      <div class="flex flex-col gap-2">
        <h2 class="text-base font-semibold text-neutral-900">Funding</h2>
        <p class="text-xs text-neutral-600 leading-4">Provide your initial estimated general fund funding goal.</p>
      </div>

      <div class="flex flex-col gap-3">
        <label class="text-xs font-medium text-neutral-900">
          Annual Funding Goal <span class="text-negative-500">*</span>
        </label>
        <lfx-input
          :model-value="modelValue.annualFundingGoal"
          placeholder="1,000"
          class="w-48"
          @update:model-value="emit('update:modelValue', { ...modelValue, annualFundingGoal: $event as string })"
        >
          <template #prefix>
            <span class="text-sm text-neutral-400">$</span>
          </template>
        </lfx-input>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { GeneralFundFormData } from '~/types/fundraise.types';
import LfxInput from '~/components/uikit/input/input.vue';

defineProps<{
  modelValue: GeneralFundFormData;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: GeneralFundFormData): void;
}>();
</script>

<script lang="ts">
export default {
  name: 'FundraiseGeneralFundFundingSection',
};
</script>
