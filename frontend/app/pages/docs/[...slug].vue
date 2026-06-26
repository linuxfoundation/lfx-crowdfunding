<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <docs-article :slug="slug" />
</template>

<script setup lang="ts">
import { computed } from 'vue';
import DocsArticle from '~/components/modules/documentation/view/docs-article.vue';

const route = useRoute();

const slug = computed(() => {
  const parts = route.params.slug;
  return Array.isArray(parts) ? parts.join('/') : (parts ?? '');
});
</script>
