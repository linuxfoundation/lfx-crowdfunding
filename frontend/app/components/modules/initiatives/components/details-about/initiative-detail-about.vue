<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <lfx-card class="p-6 flex flex-col gap-8">
    <p class="text-base font-semibold text-neutral-900 leading-6">About this initiative</p>

    <p class="text-sm text-neutral-900 leading-5">
      {{ initiative.description }}
    </p>
  </lfx-card>
</template>

<script setup lang="ts">
import LfxCard from '~/components/uikit/card/card.vue';
import type { InitiativeDetail } from '#shared/types/initiative-detail.types';

defineProps<{ initiative: InitiativeDetail }>();
</script>

<script lang="ts">
export default {
  name: 'InitiativeDetailAbout',
};
</script>
