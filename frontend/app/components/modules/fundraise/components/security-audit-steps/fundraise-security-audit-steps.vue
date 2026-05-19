<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="flex flex-col">
    <!-- Step indicator -->
    <div class="pb-6 border-b border-neutral-200">
      <fundraise-step-indicator
        :steps="STEPS"
        :current-step="currentStep"
      />
    </div>

    <!-- Step body -->
    <div class="pt-8">
      <fundraise-security-audit-details-step
        v-if="currentStep === 0"
        :model-value="modelValue"
        @update:model-value="emit('update:modelValue', $event)"
      />
      <fundraise-compliance-step
        v-else-if="currentStep === 1"
        :model-value="modelValue.compliance"
        @update:model-value="emit('update:modelValue', { ...modelValue, compliance: $event })"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import FundraiseStepIndicator from '../main/fundraise-step-indicator.vue';
import FundraiseComplianceStep from '../main/fundraise-compliance-step.vue';
import FundraiseSecurityAuditDetailsStep from './fundraise-security-audit-details-step.vue';
import type { SecurityAuditFormData } from '~/types/fundraise.types';

const STEPS = ['Initiative details', 'Compliance & Terms'];

defineProps<{
  currentStep: number;
  modelValue: SecurityAuditFormData;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: SecurityAuditFormData): void;
}>();
</script>

<script lang="ts">
export default {
  name: 'FundraiseSecurityAuditSteps',
};
</script>
