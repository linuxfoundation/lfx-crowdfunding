<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <initiative-detail-view :initiative-slug="slug" />
</template>

<script setup lang="ts">
import InitiativeDetailView from '~/components/modules/initiatives/view/initiative-detail.vue';
import type { InitiativeDetail } from '#shared/types/initiative-detail.types';

const route = useRoute();
const slug = computed(() => route.params.slug as string);
const config = useRuntimeConfig();

// Server-side fetch for SEO meta — the child component uses Vue Query for UI rendering.
const { data: initiative } = await useAsyncData<InitiativeDetail>(
  `initiative-seo-${slug.value}`,
  () => $fetch<InitiativeDetail>(`/api/initiatives/${slug.value}`),
  { lazy: false },
);

const title = computed(() => initiative.value?.name ?? slug.value);
const description = computed(() => {
  const raw = initiative.value?.description ?? '';
  return raw.length > 160
    ? `${raw.slice(0, 157)}...`
    : raw || 'Support this open source initiative on LFX Crowdfunding.';
});
const ogUrl = computed(() => `${config.public.appUrl}/initiatives/${slug.value}`);
const ogImage = computed(() => initiative.value?.logoUrl ?? `${config.public.appUrl}/og-image.png`);

useHead({ title });
useSeoMeta({
  description,
  ogTitle: computed(() => `${title.value} | LFX Crowdfunding`),
  ogDescription: description,
  ogType: 'website',
  ogUrl,
  ogImage,
  twitterCard: 'summary_large_image',
  twitterTitle: computed(() => `${title.value} | LFX Crowdfunding`),
  twitterDescription: description,
  twitterImage: ogImage,
});
</script>
