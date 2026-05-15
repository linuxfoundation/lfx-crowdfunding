<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <section class="pt-21 pb-16 flex flex-col gap-5">
    <!-- Eyebrow -->
    <div class="flex items-center gap-2 text-accent-800">
      <lfx-icon
        name="chart-pie-simple"
        type="light"
        :size="20"
      />
      <span class="text-lg font-medium leading-7">Statistics</span>
    </div>

    <!-- Headline -->
    <template v-if="isLoading">
      <lfx-skeleton
        width="70%"
        height="4.5rem"
      />
    </template>
    <h1
      v-else
      class="font-secondary font-light md:text-5xl text-4xl leading-[72px] text-black"
    >
      {{ headline }}
    </h1>

    <!-- Progress bar: gray track + gradient fill stacked via CSS grid -->
    <div class="grid place-items-start w-full">
      <div class="col-start-1 row-start-1 h-1.5 w-full rounded-full bg-neutral-200" />
      <div
        class="col-start-1 row-start-1 h-1.5 rounded-full bg-gradient-to-r from-[#009aff] to-[#00d492]"
        :style="{ width: progressWidth }"
      />
    </div>

    <!-- Subtitle -->
    <template v-if="isLoading">
      <lfx-skeleton
        width="45%"
        height="1.5rem"
      />
    </template>
    <p
      v-else
      class="text-base text-neutral-900 leading-6"
    >
      Help us reach {{ goalFormatted }} to sustain the open source ecosystem
    </p>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import LfxIcon from '~/components/uikit/icon/icon.vue';
import LfxSkeleton from '~/components/uikit/skeleton/skeleton.vue';
import { formatNumberCurrency, formatNumber } from '~/utils/formatter';
import type { StatisticsOverview } from '#shared/types/statistics.types';

const props = defineProps<{
  overview: StatisticsOverview | undefined;
  isLoading: boolean;
}>();

const headline = computed(() => {
  if (!props.overview) return '';
  const raised = formatNumberCurrency(props.overview.totalRaisedCents / 100, 'USD');
  const supporters = formatNumber(props.overview.supporterCount);
  return `${raised} funds raised by ${supporters} supporters`;
});

const goalFormatted = computed(() =>
  props.overview ? formatNumberCurrency(props.overview.annualGoalCents / 100, 'USD') : '',
);

const progressWidth = computed(() => {
  if (!props.overview || props.overview.annualGoalCents === 0) return '0%';
  const pct = (props.overview.totalRaisedCents / props.overview.annualGoalCents) * 100;
  return `${Math.min(100, Math.round(pct * 100) / 100).toFixed(2)}%`;
});
</script>

<script lang="ts">
export default {
  name: 'StatisticsHeader',
};
</script>
