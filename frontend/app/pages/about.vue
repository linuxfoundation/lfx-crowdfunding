<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <about-view />
</template>

<script setup lang="ts">
import AboutView from '~/components/modules/static-pages/view/about.vue';

const title = 'About';
const description =
  'Learn about LFX Crowdfunding — a Linux Foundation platform that enables open source projects and mentorships to receive financial support from individuals and organizations worldwide.';

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
