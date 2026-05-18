<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <fundraise-initiative-details-section
    title="Project Details"
    name-label="Project name"
    :model-value="sharedValue"
    @update:model-value="onSharedUpdate"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue';
import FundraiseInitiativeDetailsSection from '../../shared/fundraise-initiative-details-section.vue';
import type { ProjectDetailsData, InitiativeDetailsData } from '~/types/fundraise.types';

const props = defineProps<{
  modelValue: ProjectDetailsData;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: ProjectDetailsData): void;
}>();

const sharedValue = computed<InitiativeDetailsData>(() => ({
  name: props.modelValue.projectName,
  elevatorPitch: props.modelValue.elevatorPitch,
  topics: props.modelValue.topics,
  websiteUrl: props.modelValue.websiteUrl,
}));

const onSharedUpdate = (updated: InitiativeDetailsData) => {
  emit('update:modelValue', {
    ...props.modelValue,
    projectName: updated.name,
    elevatorPitch: updated.elevatorPitch,
    topics: updated.topics,
    websiteUrl: updated.websiteUrl,
  });
};
</script>

<script lang="ts">
export default {
  name: 'FundraiseProjectDetailsSection',
};
</script>
