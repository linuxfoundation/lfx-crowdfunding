<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <fundraise-branding-section
    :model-value="modelValue.logoUrl"
    @update:model-value="emit('update:modelValue', { ...modelValue, logoUrl: $event })"
  />
</template>

<script setup lang="ts">
import FundraiseBrandingSection from '../../shared/fundraise-branding-section.vue';
import type { ProjectDetailsData } from '~/types/fundraise.types';

defineProps<{
  modelValue: ProjectDetailsData;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: ProjectDetailsData): void;
}>();
</script>

<script lang="ts">
export default {
  name: 'FundraiseProjectBrandingSection',
};
</script>
