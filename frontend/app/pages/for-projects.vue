<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <for-projects-view />
</template>

<script setup lang="ts">
import ForProjectsView from '~/components/modules/static-pages/view/for-projects.vue';

const title = 'For Projects';
const description =
  'Launch a crowdfunding campaign for your open source project. Receive donations and recurring sponsorships from individuals and companies that depend on your work.';

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
