<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="md:pb-30 pb-20">
    <landing-hero />
    <landing-initiatives
      :initiatives="initiatives"
      :is-loading="isLoading"
      :error="initiativeError"
    />
    <landing-impact-stories />
    <landing-nav-cards />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import LandingHero from '../components/landing-hero.vue';
import LandingInitiatives from '../components/landing-initiatives.vue';
import LandingImpactStories from '../components/landing-impact-stories.vue';
import LandingNavCards from '../components/landing-nav-cards.vue';
import { useInitiatives } from '~/composables/initiatives/useInitiatives';
import { DEFAULT_SORT_OPTION } from '~/components/modules/initiatives/config/initiatives-header.config';

const { data, isLoading, error } = useInitiatives({ pageSize: 3, sortBy: DEFAULT_SORT_OPTION.value, sortDir: 'desc' });
const initiatives = computed(() => data.value?.pages.flatMap((p) => p.data) ?? []);
const initiativeError = computed(() => error.value as Error | null);
</script>

<script lang="ts">
export default {
  name: 'LandingView',
};
</script>
