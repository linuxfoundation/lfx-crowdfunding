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
import FundraiseProjectHostingStep from './fundraise-project-hosting-step.vue';
import FundraiseProjectGithubStep from './github-sub-steps/fundraise-project-github-step.vue';
import FundraiseProjectDetailsStep from './details/fundraise-project-details-step.vue';
import type { ProjectFormData } from '~/types/fundraise.types';

const STEPS_GITHUB = ['Project hosting', 'Connect GitHub', 'Initiative details', 'Compliance & Terms'];
const STEPS_DEFAULT = ['Project hosting', 'Initiative details', 'Compliance & Terms'];

const props = defineProps<{
  currentStep: number;
  modelValue: ProjectFormData;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: ProjectFormData): void;
}>();

const steps = computed(() => (props.modelValue.hostingType === 'github' ? STEPS_GITHUB : STEPS_DEFAULT));

const detailsStepIndex = computed(() => (props.modelValue.hostingType === 'github' ? 2 : 1));
const complianceStepIndex = computed(() => detailsStepIndex.value + 1);
</script>

<script lang="ts">
export default {
  name: 'FundraiseProjectSteps',
};
</script>
