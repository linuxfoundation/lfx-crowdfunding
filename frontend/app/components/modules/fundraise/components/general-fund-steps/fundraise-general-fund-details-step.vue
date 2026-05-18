<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="flex flex-col gap-4">
    <fundraise-initiative-details-section
      title="General Fund Details"
      name-label="General Fund name"
      :model-value="initiativeDetails"
      @update:model-value="onDetailsUpdate"
    />
    <fundraise-branding-section
      :model-value="modelValue.logoFileName"
      @update:model-value="emit('update:modelValue', { ...modelValue, logoFileName: $event })"
    />
    <fundraise-beneficiaries-section
      :model-value="modelValue.beneficiaries"
      @update:model-value="emit('update:modelValue', { ...modelValue, beneficiaries: $event })"
    />
    <fundraise-general-fund-funding-section
      :model-value="modelValue"
      @update:model-value="emit('update:modelValue', $event)"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import FundraiseInitiativeDetailsSection from '../shared/fundraise-initiative-details-section.vue';
import FundraiseBrandingSection from '../shared/fundraise-branding-section.vue';
import FundraiseBeneficiariesSection from '../shared/fundraise-beneficiaries-section.vue';
import FundraiseGeneralFundFundingSection from './sections/fundraise-general-fund-funding-section.vue';
import type { GeneralFundFormData, InitiativeDetailsData } from '~/types/fundraise.types';

const props = defineProps<{
  modelValue: GeneralFundFormData;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: GeneralFundFormData): void;
}>();

const initiativeDetails = computed<InitiativeDetailsData>(() => ({
  name: props.modelValue.name,
  elevatorPitch: props.modelValue.elevatorPitch,
  topics: props.modelValue.topics,
  websiteUrl: props.modelValue.websiteUrl,
}));

const onDetailsUpdate = (updated: InitiativeDetailsData) => {
  emit('update:modelValue', {
    ...props.modelValue,
    name: updated.name,
    elevatorPitch: updated.elevatorPitch,
    topics: updated.topics,
    websiteUrl: updated.websiteUrl,
  });
};
</script>

<script lang="ts">
export default {
  name: 'FundraiseGeneralFundDetailsStep',
};
</script>
