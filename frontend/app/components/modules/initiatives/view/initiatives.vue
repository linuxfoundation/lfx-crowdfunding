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
    <div
      class="transition-all ease-linear"
      :class="{ 'pt-8': isScrolled }"
    >
      <initiatives-grid
        :initiatives="data?.data ?? []"
        :is-loading="isLoading"
        :error="initiativeError"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import InitiativesHeader from '../components/initiatives-header.vue';
import InitiativesGrid from '../components/initiatives-grid.vue';
import { useInitiatives } from '~/composables/initiatives/useInitiatives';
import useScroll from '~/utils/scroll';

const { scrollTop } = useScroll();

const searchTerm = ref('');
const activeType = ref('all');
const sortBy = ref('recent');

const { data, isLoading, error } = useInitiatives({
  search: searchTerm,
  type: activeType,
  sort: sortBy,
});

const initiativeError = computed(() => error.value as Error | null);

const isScrolled = computed(() => scrollTop.value > 10);
</script>

<script lang="ts">
export default {
  name: 'InitiativesView',
};
</script>
