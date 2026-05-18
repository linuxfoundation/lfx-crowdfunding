<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <fundraise-fund-distribution-section
    title="Event Budget"
    description="Provide your initial estimated event budget. You can update your goal at any time and continue raising funds after your goal is met."
    goal-label="Sponsorship Goal"
    distribution-label="Budget distribution"
    :model-value="distributionData"
    @update:model-value="onUpdate"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue';
import FundraiseFundDistributionSection from '../../shared/fundraise-fund-distribution-section.vue';
import type { EventFormData, FundDistributionData } from '~/types/fundraise.types';

const props = defineProps<{
  modelValue: EventFormData;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: EventFormData): void;
}>();

const distributionData = computed<FundDistributionData>(() => ({
  goal: props.modelValue.sponsorshipGoal,
  distribution: props.modelValue.budgetDistribution,
}));

const onUpdate = (updated: FundDistributionData) => {
  emit('update:modelValue', {
    ...props.modelValue,
    sponsorshipGoal: updated.goal,
    budgetDistribution: updated.distribution,
  });
};
</script>

<script lang="ts">
export default {
  name: 'FundraiseEventBudgetSection',
};
</script>
