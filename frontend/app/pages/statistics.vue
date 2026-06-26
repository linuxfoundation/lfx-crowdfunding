<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <statistics-view />
</template>

<script setup lang="ts">
import StatisticsView from '~/components/modules/statistics/view/statistics.vue';

const title = 'Statistics';
const description =
  'Explore real-time funding statistics across all open source initiatives on LFX Crowdfunding — total donations, active supporters, and funded projects.';

useHead({ title });
useSeoMeta({
  description,
  ogTitle: `${title} | LFX Crowdfunding`,
  ogDescription: description,
  ogType: 'website',
  twitterCard: 'summary_large_image',
  twitterTitle: `${title} | LFX Crowdfunding`,
  twitterDescription: description,
});
</script>
