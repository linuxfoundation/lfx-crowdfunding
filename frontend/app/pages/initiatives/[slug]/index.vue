<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <initiative-detail-view :initiative-slug="slug" />
</template>

<script setup lang="ts">
import InitiativeDetailView from '~/components/modules/initiatives/view/initiative-detail.vue';

const route = useRoute();
const slug = computed(() => route.params.slug as string);

useHead({ title: computed(() => slug.value) });
</script>
