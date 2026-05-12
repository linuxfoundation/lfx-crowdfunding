<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <initiative-detail-view :initiative-id="id" />
</template>

<script setup lang="ts">
import InitiativeDetailView from '~/components/modules/initiatives/view/initiative-detail.vue';

const route = useRoute();
const id = computed(() => route.params.id as string);

useHead({ title: computed(() => id.value) });
</script>
