<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <fundraise-fund-distribution-section
    title="Funding"
    description="Provide your initial estimated annual project budget. You can update your goal at any time and continue raising funds after your goal is met."
    goal-label="Annual Funding Goal"
    :model-value="distributionData"
    @update:model-value="onUpdate"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue';
import FundraiseFundDistributionSection from '../../shared/fundraise-fund-distribution-section.vue';
import type { ProjectDetailsData, FundDistributionData } from '~/types/fundraise.types';

const props = defineProps<{
  modelValue: ProjectDetailsData;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: ProjectDetailsData): void;
}>();

const distributionData = computed<FundDistributionData>(() => ({
  goal: props.modelValue.annualFundingGoal,
  distribution: props.modelValue.goals,
}));

const onUpdate = (updated: FundDistributionData) => {
  emit('update:modelValue', {
    ...props.modelValue,
    annualFundingGoal: updated.goal,
    goals: updated.distribution,
  });
};
</script>

<script lang="ts">
export default {
  name: 'FundraiseProjectFundingSection',
};
</script>
