<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="flex flex-col gap-8">
    <initiative-detail-funding-card :initiative="initiative" />

    <initiative-detail-impact
      v-if="initiative.impactStats?.length"
      :stats="initiative.impactStats"
    />

    <initiative-detail-project-health
      v-if="initiative.projectHealthStats?.length"
      :stats="initiative.projectHealthStats"
      :rating="initiative.projectHealthRating"
    />
  </div>
</template>

<script setup lang="ts">
import InitiativeDetailFundingCard from './initiative-detail-funding-card.vue';
import InitiativeDetailImpact from './initiative-detail-impact.vue';
import InitiativeDetailProjectHealth from './initiative-detail-project-health.vue';
import type { InitiativeDetail } from '#shared/types/initiative-detail.types';

defineProps<{ initiative: InitiativeDetail }>();
</script>

<script lang="ts">
export default {
  name: 'InitiativeDetailOverview',
};
</script>
