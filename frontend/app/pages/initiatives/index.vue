<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <initiatives-view />
</template>

<script setup lang="ts">
import InitiativesView from '~/components/modules/initiatives/view/initiatives.vue';

const title = 'Initiatives';
const description =
  'Browse all open source initiatives on LFX Crowdfunding. Support projects, mentorships, events, and security research from the Linux Foundation ecosystem.';

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
