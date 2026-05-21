<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <lfx-card class="p-6 flex flex-col gap-6">
    <p class="text-base font-semibold text-neutral-900 leading-6">Monthly donations</p>

    <template v-if="isLoading">
      <div class="flex items-start justify-between gap-4">
        <div class="flex-1 flex flex-col gap-1">
          <lfx-skeleton
            height="3.5rem"
            width="6rem"
          />
          <lfx-skeleton
            height="1.25rem"
            width="8rem"
          />
        </div>
        <div class="flex-1 flex flex-col gap-1">
          <lfx-skeleton
            height="3.5rem"
            width="4rem"
          />
          <lfx-skeleton
            height="1.25rem"
            width="7rem"
          />
        </div>
        <lfx-skeleton
          class="flex-1"
          height="5rem"
        />
      </div>
    </template>

    <div
      v-else-if="error"
      class="flex items-center gap-2 text-negative-600"
    >
      <lfx-icon
        name="circle-exclamation"
        type="solid"
        :size="16"
      />
      <span class="text-sm">Failed to load monthly donations.</span>
    </div>

    <div
      v-else-if="monthly"
      class="flex items-start justify-between md:gap-4 gap-8 md:flex-row flex-col"
    >
      <!-- Left: raised amount for most recent month -->
      <div class="flex-1 w-full flex flex-col min-w-0">
        <p class="text-4xl font-normal leading-[56px] text-neutral-900 whitespace-nowrap">
          {{ totalFormatted }}
        </p>
        <div class="flex flex-col gap-2">
          <p class="text-sm text-neutral-600">Raised in {{ latestPeriodLabel }}</p>
        </div>
      </div>

      <!-- Middle: supporters for most recent month -->
      <div class="flex-1 w-full flex flex-col min-w-0">
        <p class="text-4xl font-normal leading-[56px] text-neutral-900">
          {{ latestSupporters }}
        </p>
        <p class="text-sm text-neutral-600">New supporters</p>
      </div>

      <!-- Right: bar chart (all 12 buckets) -->
      <statistics-monthly-bar-chart :buckets="activeBuckets" />
    </div>
  </lfx-card>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import StatisticsMonthlyBarChart from './statistics-monthly-bar-chart.vue';
import LfxCard from '~/components/uikit/card/card.vue';
import LfxIcon from '~/components/uikit/icon/icon.vue';
import LfxSkeleton from '~/components/uikit/skeleton/skeleton.vue';
import { formatNumberCurrency } from '~/utils/formatter';
import type { MonthlyDonations } from '#shared/types/statistics.types';

const MONTH_ABBR = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];

const props = defineProps<{
  monthly: MonthlyDonations | undefined;
  isLoading: boolean;
  error: Error | null;
}>();

const activeBuckets = computed(() => {
  const buckets = props.monthly?.buckets ?? [];
  const lastNonZero = buckets.reduce((acc, b, i) => (b.totalCents > 0 ? i : acc), -1);
  return lastNonZero >= 0 ? buckets.slice(0, lastNonZero + 1) : buckets;
});

const latestBucket = computed(() => {
  const buckets = activeBuckets.value;
  return buckets.length > 0 ? buckets[buckets.length - 1] : undefined;
});

const totalFormatted = computed(() =>
  latestBucket.value ? formatNumberCurrency(latestBucket.value.totalCents / 100, 'USD') : '',
);

const latestPeriodLabel = computed(() => {
  const b = latestBucket.value;
  return b ? `${MONTH_ABBR[b.month - 1]} ${b.year}` : '';
});

const latestSupporters = computed(() => latestBucket.value?.supporters ?? 0);
</script>

<script lang="ts">
export default {
  name: 'StatisticsMonthlyDonations',
};
</script>
