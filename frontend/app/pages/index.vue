<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <landing-view />
</template>

<script setup lang="ts">
import LandingView from '~/components/modules/landing/view/landing.vue';

const title = 'Home';
const description =
  'LFX Crowdfunding connects open source contributors, projects, and organizations. Fund the projects that power the world.';

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
