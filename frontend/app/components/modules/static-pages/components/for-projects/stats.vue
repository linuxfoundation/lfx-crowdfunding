<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="border-t border-neutral-200 py-10 flex gap-8 items-start">
    <template v-if="isLoading">
      <div
        v-for="n in 4"
        :key="n"
        class="flex-1 flex flex-col gap-2"
      >
        <lfx-skeleton
          height="3.5rem"
          width="8rem"
        />
        <lfx-skeleton
          height="1.5rem"
          width="10rem"
        />
      </div>
    </template>

    <template v-else>
      <div class="flex-1 flex flex-col gap-2.5 min-w-0">
        <p class="text-4xl leading-14 text-neutral-900">{{ totalRaised }}</p>
        <p class="text-base leading-6 text-neutral-900">Total Funds Raised</p>
      </div>
      <div class="flex-1 flex flex-col gap-2.5 min-w-0">
        <p class="text-4xl leading-14 text-neutral-900">{{ activeInitiatives }}</p>
        <p class="text-base leading-6 text-neutral-900">Active Initiatives</p>
      </div>
      <div class="flex-1 flex flex-col gap-2.5 min-w-0">
        <p class="text-4xl leading-14 text-neutral-900">{{ supporters }}</p>
        <p class="text-base leading-6 text-neutral-900">Unique Supporters</p>
      </div>
      <div class="flex-1 flex flex-col gap-2.5 min-w-0">
        <p class="text-4xl leading-14 text-neutral-900">0%</p>
        <p class="text-base leading-6 text-neutral-900">Platform Fees</p>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import LfxSkeleton from '~/components/uikit/skeleton/skeleton.vue';
import { formatNumberCurrency, formatNumber } from '~/utils/formatter';
import type { StatisticsOverview } from '#shared/types/statistics.types';

const props = defineProps<{
  overview: StatisticsOverview | undefined;
  isLoading: boolean;
}>();

const totalRaised = computed(() =>
  props.overview ? formatNumberCurrency(props.overview.totalRaisedCents / 100, 'USD') + '+' : '',
);

const activeInitiatives = computed(() => (props.overview ? String(props.overview.activeInitiatives) : ''));

const supporters = computed(() => (props.overview ? formatNumber(props.overview.supporterCount) + '+' : ''));
</script>

<script lang="ts">
export default {
  name: 'ForProjectsStats',
};
</script>
