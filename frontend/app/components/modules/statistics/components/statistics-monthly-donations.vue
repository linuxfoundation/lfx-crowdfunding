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
      <!-- Left: raised amount + label + change indicator -->
      <div class="flex-1 w-full flex flex-col min-w-0">
        <p class="text-4xl font-normal leading-[56px] text-neutral-900 whitespace-nowrap">
          {{ totalFormatted }}
        </p>
        <div class="flex flex-col gap-2">
          <p class="text-sm text-neutral-600">Raised in {{ monthly.periodLabel }}</p>
          <div class="flex items-center gap-1">
            <lfx-icon
              name="circle-arrow-up"
              type="solid"
              :size="12"
              class="text-positive-600"
            />
            <span class="text-[10px] leading-[14px] font-medium text-positive-600">+{{ monthly.percentChange }}%</span>
            <span class="text-[10px] leading-[14px] text-neutral-600">vs. last month</span>
          </div>
        </div>
      </div>

      <!-- Middle: new supporters -->
      <div class="flex-1 w-full flex flex-col min-w-0">
        <p class="text-4xl font-normal leading-[56px] text-neutral-900">
          {{ monthly.newSupporters }}
        </p>
        <p class="text-sm text-neutral-600">New supporters</p>
      </div>

      <!-- Right: mini bar chart -->
      <statistics-monthly-bar-chart :daily="monthly.daily" />
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

const props = defineProps<{
  monthly: MonthlyDonations | undefined;
  isLoading: boolean;
  error: Error | null;
}>();

const totalFormatted = computed(() =>
  props.monthly ? formatNumberCurrency(props.monthly.totalCents / 100, 'USD') : '',
);
</script>

<script lang="ts">
export default {
  name: 'StatisticsMonthlyDonations',
};
</script>
