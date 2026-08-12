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
      <fundraise-event-details-step
        v-if="currentStep === 0"
        :model-value="modelValue"
        @update:model-value="emit('update:modelValue', $event)"
      />
      <fundraise-donation-options-step
        v-else-if="currentStep === donationOptionsStepIndex"
        :model-value="modelValue.donationOptions"
        @update:model-value="emit('update:modelValue', { ...modelValue, donationOptions: $event })"
      />
      <fundraise-attribution-step
        v-else-if="props.showAttribution && currentStep === attributionStepIndex"
        :model-value="modelValue.attribution"
        @update:model-value="emit('update:modelValue', { ...modelValue, attribution: $event })"
      />
      <fundraise-compliance-step
        v-else-if="currentStep === complianceStepIndex"
        :model-value="modelValue.compliance"
        @update:model-value="emit('update:modelValue', { ...modelValue, compliance: $event })"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import FundraiseStepIndicator from '../main/fundraise-step-indicator.vue';
import FundraiseComplianceStep from '../main/fundraise-compliance-step.vue';
import FundraiseDonationOptionsStep from '../main/fundraise-donation-options-step.vue';
import FundraiseAttributionStep from '../main/fundraise-attribution-step.vue';
import FundraiseEventDetailsStep from './fundraise-event-details-step.vue';
import type { EventFormData } from '~/types/fundraise.types';

const props = defineProps<{
  currentStep: number;
  modelValue: EventFormData;
  showAttribution: boolean;
}>();

const donationOptionsStepIndex = 1;
const attributionStepIndex = donationOptionsStepIndex + 1;
const complianceStepIndex = computed(() =>
  props.showAttribution ? attributionStepIndex + 1 : donationOptionsStepIndex + 1,
);
const STEPS = computed(() =>
  props.showAttribution
    ? ['Initiative details', 'Donation options', 'Attribution', 'Compliance & Terms']
    : ['Initiative details', 'Donation options', 'Compliance & Terms'],
);

const emit = defineEmits<{
  (e: 'update:modelValue', value: EventFormData): void;
}>();
</script>

<script lang="ts">
export default {
  name: 'FundraiseEventSteps',
};
</script>
