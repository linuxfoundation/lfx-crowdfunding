<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <docs-landing />
</template>

<script setup lang="ts">
import DocsLanding from '~/components/modules/documentation/view/docs-landing.vue';

const url = useRequestURL();

useHead({ title: 'Documentation' });
useSeoMeta({
  description: 'Learn how to use LFX Crowdfunding — browse initiatives, make donations, manage your account, and more.',
  ogTitle: 'Documentation — LFX Crowdfunding',
  ogDescription:
    'Learn how to use LFX Crowdfunding — browse initiatives, make donations, manage your account, and more.',
  ogType: 'website',
  ogUrl: url.href,
  twitterCard: 'summary',
  twitterTitle: 'Documentation — LFX Crowdfunding',
  twitterDescription:
    'Learn how to use LFX Crowdfunding — browse initiatives, make donations, manage your account, and more.',
});
useHead({ link: [{ rel: 'canonical', href: url.href }] });
</script>
