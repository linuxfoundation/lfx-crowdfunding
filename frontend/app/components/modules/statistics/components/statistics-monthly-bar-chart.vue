<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div
    class="flex-1 w-full flex flex-col min-w-0"
    :aria-label="
      props.buckets.length === 0
        ? 'Monthly donations bar chart. No data available.'
        : `Monthly donations bar chart. Peak month: ${peakLabel} with ${peakAmount}.`
    "
    role="img"
  >
    <Bar
      :data="chartData"
      :options="chartOptions"
      class="w-full"
      style="height: 90px"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { Bar } from 'vue-chartjs';
import { Chart as ChartJS, BarController, BarElement, CategoryScale, LinearScale, Tooltip } from 'chart.js';
import { formatNumberCurrency } from '~/utils/formatter';
import type { MonthlyBucket } from '#shared/types/statistics.types';

ChartJS.register(BarController, BarElement, CategoryScale, LinearScale, Tooltip);

const MONTH_ABBR = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];

const props = defineProps<{
  buckets: MonthlyBucket[];
}>();

const peakIndex = computed(() => {
  let max = -1;
  let idx = 0;
  props.buckets.forEach((b, i) => {
    if (b.totalCents > max) {
      max = b.totalCents;
      idx = i;
    }
  });
  return idx;
});

const peakLabel = computed(() => {
  const b = props.buckets[peakIndex.value];
  return b ? `${MONTH_ABBR[b.month - 1]} ${b.year}` : '';
});
const peakAmount = computed(() =>
  props.buckets[peakIndex.value] ? formatNumberCurrency(props.buckets[peakIndex.value].totalCents / 100, 'USD') : '',
);

const NORMAL_COLOR = '#b8d9ff';
const HIGHLIGHT_COLOR = '#009aff';

const chartData = computed(() => ({
  labels: props.buckets.map((b, i) => {
    if (i === 0 || i === props.buckets.length - 1) return MONTH_ABBR[b.month - 1];
    return '';
  }),
  datasets: [
    {
      data: props.buckets.map((b) => b.totalCents / 100),
      backgroundColor: props.buckets.map((_, i) => (i === peakIndex.value ? HIGHLIGHT_COLOR : NORMAL_COLOR)),
      borderRadius: 4,
      borderSkipped: false,
      barPercentage: 0.8,
      categoryPercentage: 0.9,
    },
  ],
}));

const chartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: { display: false },
    tooltip: {
      callbacks: {
        label: (ctx: { raw: number }) => formatNumberCurrency(ctx.raw, 'USD'),
      },
    },
  },
  scales: {
    x: {
      grid: { display: false },
      border: { display: false },
      ticks: {
        color: '#45556c',
        font: { size: 10 },
        maxRotation: 0,
        autoSkip: false,
      },
    },
    y: {
      display: false,
      grid: { display: false },
    },
  },
};
</script>

<script lang="ts">
export default {
  name: 'StatisticsMonthlyBarChart',
};
</script>
