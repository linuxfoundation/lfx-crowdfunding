<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="flex flex-col gap-4">
    <fundraise-event-details-section
      :model-value="modelValue"
      @update:model-value="emit('update:modelValue', $event)"
    />
    <fundraise-event-date-location-section
      :model-value="modelValue"
      @update:model-value="emit('update:modelValue', $event)"
    />
    <fundraise-branding-section
      :model-value="modelValue.logoFileName"
      @update:model-value="emit('update:modelValue', { ...modelValue, logoFileName: $event })"
    />
    <fundraise-beneficiaries-section
      :model-value="modelValue.beneficiaries"
      @update:model-value="emit('update:modelValue', { ...modelValue, beneficiaries: $event })"
    />
    <fundraise-event-budget-section
      :model-value="modelValue"
      @update:model-value="emit('update:modelValue', $event)"
    />
    <fundraise-event-funding-goals-section />
    <fundraise-event-sponsorship-tiers-section />
  </div>
</template>

<script setup lang="ts">
import FundraiseBrandingSection from '../shared/fundraise-branding-section.vue';
import FundraiseBeneficiariesSection from '../shared/fundraise-beneficiaries-section.vue';
import FundraiseEventDetailsSection from './sections/fundraise-event-details-section.vue';
import FundraiseEventDateLocationSection from './sections/fundraise-event-date-location-section.vue';
import FundraiseEventBudgetSection from './sections/fundraise-event-budget-section.vue';
import FundraiseEventFundingGoalsSection from './sections/fundraise-event-funding-goals-section.vue';
import FundraiseEventSponsorshipTiersSection from './sections/fundraise-event-sponsorship-tiers-section.vue';
import type { EventFormData } from '~/types/fundraise.types';

defineProps<{
  modelValue: EventFormData;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: EventFormData): void;
}>();
</script>

<script lang="ts">
export default {
  name: 'FundraiseEventDetailsStep',
};
</script>
