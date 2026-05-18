<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="flex flex-col gap-4">
    <fundraise-project-details-section
      :model-value="modelValue"
      @update:model-value="$emit('update:modelValue', $event)"
    />
    <fundraise-project-compliance-section
      :model-value="modelValue"
      @update:model-value="$emit('update:modelValue', $event)"
    />
    <fundraise-project-branding-section
      :model-value="modelValue"
      @update:model-value="$emit('update:modelValue', $event)"
    />
    <fundraise-project-beneficiaries-section
      :model-value="modelValue"
      @update:model-value="$emit('update:modelValue', $event)"
    />
    <fundraise-project-funding-section
      :model-value="modelValue"
      @update:model-value="$emit('update:modelValue', $event)"
    />
  </div>
</template>

<script setup lang="ts">
import FundraiseProjectDetailsSection from './fundraise-project-details-section.vue';
import FundraiseProjectComplianceSection from './fundraise-project-compliance-section.vue';
import FundraiseProjectBrandingSection from './fundraise-project-branding-section.vue';
import FundraiseProjectBeneficiariesSection from './fundraise-project-beneficiaries-section.vue';
import FundraiseProjectFundingSection from './fundraise-project-funding-section.vue';
import type { ProjectDetailsData } from '~/types/fundraise.types';

defineProps<{
  modelValue: ProjectDetailsData;
}>();

defineEmits<{
  (e: 'update:modelValue', value: ProjectDetailsData): void;
}>();
</script>

<script lang="ts">
export default {
  name: 'FundraiseProjectDetailsStep',
};
</script>
