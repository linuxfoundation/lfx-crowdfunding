<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div>
    <landing-hero />
    <landing-initiatives
      :initiatives="data?.data ?? []"
      :is-loading="isLoading"
      :error="initiativeError"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import LandingHero from '../components/landing-hero.vue';
import LandingInitiatives from '../components/landing-initiatives.vue';
import { useInitiatives } from '~/composables/useInitiatives';

const { data, isLoading, error } = useInitiatives();
const initiativeError = computed(() => error.value as Error | null);
</script>

<script lang="ts">
export default {
  name: 'LandingView',
};
</script>
