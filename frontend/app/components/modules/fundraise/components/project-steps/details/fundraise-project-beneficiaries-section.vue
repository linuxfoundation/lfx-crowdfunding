<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <fundraise-beneficiaries-section
    :model-value="modelValue.beneficiaries"
    @update:model-value="emit('update:modelValue', { ...modelValue, beneficiaries: $event })"
  />
</template>

<script setup lang="ts">
import FundraiseBeneficiariesSection from '../../shared/fundraise-beneficiaries-section.vue';
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
  name: 'FundraiseProjectBeneficiariesSection',
};
</script>
