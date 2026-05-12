<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <lfx-card class="p-6 flex flex-col gap-8">
    <p class="text-base font-semibold text-neutral-900 leading-6">Impact</p>

    <div class="flex gap-4 items-start">
      <div
        v-for="stat in stats"
        :key="stat.label"
        class="flex-1 min-w-0 flex flex-col gap-1"
      >
        <p class="text-4xl text-neutral-900 leading-[56px]">
          {{ stat.value }}
        </p>
        <p class="text-sm text-neutral-900 leading-5">
          {{ stat.label }}
        </p>
      </div>
    </div>
  </lfx-card>
</template>

<script setup lang="ts">
import type { ImpactStat } from '#shared/types/initiative-detail.types';
import LfxCard from '~/components/uikit/card/card.vue';

defineProps<{ stats: ImpactStat[] }>();
</script>

<script lang="ts">
export default {
  name: 'InitiativeDetailImpact',
};
</script>
