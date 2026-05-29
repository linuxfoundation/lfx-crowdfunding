<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="flex flex-col gap-4">
    <fundraise-initiative-details-section
      title="OSTIF Security Audit details"
      name-label="OSTIF Security Audit name"
      :show-repository-url="true"
      :model-value="initiativeDetails"
      @update:model-value="onDetailsUpdate"
    />
    <fundraise-security-details-section
      :model-value="modelValue"
      @update:model-value="emit('update:modelValue', $event)"
    />
    <fundraise-governance-section
      :model-value="modelValue"
      @update:model-value="emit('update:modelValue', $event)"
    />
    <fundraise-branding-section
      :model-value="modelValue.logoUrl"
      @update:model-value="emit('update:modelValue', { ...modelValue, logoUrl: $event })"
    />
    <fundraise-contact-section
      :model-value="modelValue"
      @update:model-value="emit('update:modelValue', $event)"
    />
    <fundraise-security-funding-section
      :model-value="modelValue"
      @update:model-value="emit('update:modelValue', $event)"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import FundraiseInitiativeDetailsSection from '../shared/fundraise-initiative-details-section.vue';
import FundraiseBrandingSection from '../shared/fundraise-branding-section.vue';
import FundraiseSecurityDetailsSection from './sections/fundraise-security-details-section.vue';
import FundraiseGovernanceSection from './sections/fundraise-governance-section.vue';
import FundraiseContactSection from './sections/fundraise-contact-section.vue';
import FundraiseSecurityFundingSection from './sections/fundraise-security-funding-section.vue';
import type { SecurityAuditFormData, InitiativeDetailsData } from '~/types/fundraise.types';

const props = defineProps<{
  modelValue: SecurityAuditFormData;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: SecurityAuditFormData): void;
}>();

const initiativeDetails = computed<InitiativeDetailsData>(() => ({
  name: props.modelValue.auditName,
  elevatorPitch: props.modelValue.elevatorPitch,
  topics: props.modelValue.topics,
  repositoryUrl: props.modelValue.repositoryUrl,
  websiteUrl: props.modelValue.websiteUrl,
}));

const onDetailsUpdate = (updated: InitiativeDetailsData) => {
  emit('update:modelValue', {
    ...props.modelValue,
    auditName: updated.name,
    elevatorPitch: updated.elevatorPitch,
    topics: updated.topics,
    repositoryUrl: updated.repositoryUrl ?? '',
    websiteUrl: updated.websiteUrl,
  });
};
</script>

<script lang="ts">
export default {
  name: 'FundraiseSecurityAuditDetailsStep',
};
</script>
