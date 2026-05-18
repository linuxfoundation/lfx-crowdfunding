<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="border border-neutral-200 rounded-xl p-6">
    <div class="flex flex-col gap-5">
      <div class="flex flex-col gap-2">
        <h2 class="text-base font-semibold text-neutral-900">Funding</h2>
        <p class="text-xs text-neutral-600 leading-4">
          The funding goal amount will be determined by OSTIF when they review and approve your project.
        </p>
      </div>

      <div class="flex flex-col gap-3">
        <label class="text-xs font-medium text-neutral-900">
          Security Audit Funding Goal <span class="text-negative-500">*</span>
        </label>
        <lfx-input
          :model-value="modelValue.fundingGoal"
          placeholder=""
          class="w-48"
          @update:model-value="emit('update:modelValue', { ...modelValue, fundingGoal: $event as string })"
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
import type { SecurityAuditFormData } from '~/types/fundraise.types';
import LfxInput from '~/components/uikit/input/input.vue';

defineProps<{
  modelValue: SecurityAuditFormData;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: SecurityAuditFormData): void;
}>();
</script>

<script lang="ts">
export default {
  name: 'FundraiseSecurityFundingSection',
};
</script>
