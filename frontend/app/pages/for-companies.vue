<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <for-companies-view />
</template>

<script setup lang="ts">
import ForCompaniesView from '~/components/modules/static-pages/view/for-companies.vue';

const title = 'For Companies';
const description =
  'Invest in the open source projects that power your business. Sponsor initiatives on LFX Crowdfunding and demonstrate your commitment to the open source ecosystem.';

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
