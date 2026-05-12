<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div>
    <initiatives-header
      v-model:search-term="searchTerm"
      v-model:active-type="activeType"
      v-model:sort-by="sortBy"
    />
    <initiatives-grid
      :initiatives="data?.data ?? []"
      :is-loading="isLoading"
      :error="initiativeError"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import InitiativesHeader from '../components/initiatives-header.vue';
import InitiativesGrid from '../components/initiatives-grid.vue';
import { useInitiatives } from '~/composables/useInitiatives';

const searchTerm = ref('');
const activeType = ref('all');
const sortBy = ref('recent');

const { data, isLoading, error } = useInitiatives({
  search: searchTerm,
  type: activeType,
  sort: sortBy,
});

const initiativeError = computed(() => error.value as Error | null);
</script>

<script lang="ts">
export default {
  name: 'InitiativesView',
};
</script>
