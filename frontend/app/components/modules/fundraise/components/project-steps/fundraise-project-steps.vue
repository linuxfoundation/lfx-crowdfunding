<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="flex flex-col">
    <!-- Step indicator -->
    <div class="pb-6 border-b border-neutral-200">
      <fundraise-step-indicator
        :steps="steps"
        :current-step="currentStep"
      />
    </div>

    <!-- Step body -->
    <div class="pt-8">
      <fundraise-project-hosting-step
        v-if="currentStep === 0"
        :model-value="modelValue.hostingType"
        @update:model-value="emit('update:modelValue', { ...modelValue, hostingType: $event })"
      />

      <fundraise-project-github-step
        v-else-if="currentStep === 1 && modelValue.hostingType === 'github'"
        :model-value="modelValue.selectedRepo"
        @update:model-value="emit('update:modelValue', { ...modelValue, selectedRepo: $event })"
      />

      <!-- TODO: step (git_url path) - fundraise-project-git-url-step -->

      <fundraise-project-details-step
        v-else-if="currentStep === detailsStepIndex"
        :model-value="modelValue.details"
        :show-repository-url="modelValue.hostingType !== 'github'"
        @update:model-value="emit('update:modelValue', { ...modelValue, details: $event })"
      />

      <fundraise-donation-options-step
        v-else-if="currentStep === donationOptionsStepIndex"
        :model-value="modelValue.donationOptions"
        @update:model-value="emit('update:modelValue', { ...modelValue, donationOptions: $event })"
      />

      <fundraise-attribution-step
        v-else-if="showAttribution && currentStep === attributionStepIndex"
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
import FundraiseProjectHostingStep from './fundraise-project-hosting-step.vue';
import FundraiseProjectGithubStep from './github-sub-steps/fundraise-project-github-step.vue';
import FundraiseProjectDetailsStep from './details/fundraise-project-details-step.vue';
import type { ProjectFormData } from '~/types/fundraise.types';

const props = defineProps<{
  currentStep: number;
  modelValue: ProjectFormData;
  showAttribution: boolean;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: ProjectFormData): void;
}>();

const detailsStepIndex = computed(() => (props.modelValue.hostingType === 'github' ? 2 : 1));
const donationOptionsStepIndex = computed(() => detailsStepIndex.value + 1);
const attributionStepIndex = computed(() => donationOptionsStepIndex.value + 1);
const complianceStepIndex = computed(() =>
  props.showAttribution ? attributionStepIndex.value + 1 : donationOptionsStepIndex.value + 1,
);

const steps = computed(() => {
  const base =
    props.modelValue.hostingType === 'github'
      ? ['Project hosting', 'Connect GitHub', 'Initiative details', 'Donation options']
      : ['Project hosting', 'Initiative details', 'Donation options'];
  return props.showAttribution ? [...base, 'Attribution', 'Compliance & Terms'] : [...base, 'Compliance & Terms'];
});
</script>

<script lang="ts">
export default {
  name: 'FundraiseProjectSteps',
};
</script>
