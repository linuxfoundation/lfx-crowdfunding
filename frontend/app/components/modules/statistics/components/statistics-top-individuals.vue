<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <statistics-top-list
    title="Top Individuals"
    :entries="entries"
    entry-type="member"
    :is-loading="isLoading"
    :error="error"
  />
</template>

<script setup lang="ts">
import StatisticsTopList from './statistics-top-list.vue';
import type { TopDonor } from '#shared/types/statistics.types';

defineProps<{
  entries: TopDonor[];
  isLoading: boolean;
  error: Error | null;
}>();
</script>

<script lang="ts">
export default {
  name: 'StatisticsTopIndividuals',
};
</script>
