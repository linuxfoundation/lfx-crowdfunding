<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div
    class="flex-1 w-full flex flex-col min-w-0"
    :aria-label="`Monthly donations bar chart. Peak day: ${peakLabel} with ${peakAmount}.`"
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
import type { MonthlyDonationsDaily } from '#shared/types/statistics.types';

ChartJS.register(BarController, BarElement, CategoryScale, LinearScale, Tooltip);

const props = defineProps<{
  daily: MonthlyDonationsDaily[];
}>();

const peakIndex = computed(() => {
  let max = -1;
  let idx = 0;
  props.daily.forEach((d, i) => {
    if (d.cents > max) {
      max = d.cents;
      idx = i;
    }
  });
  return idx;
});

const peakLabel = computed(() => props.daily[peakIndex.value]?.date ?? '');
const peakAmount = computed(() =>
  props.daily[peakIndex.value] ? formatNumberCurrency(props.daily[peakIndex.value].cents / 100, 'USD') : '',
);

const NORMAL_COLOR = '#b8d9ff';
const HIGHLIGHT_COLOR = '#009aff';

const formatAxisDate = (isoDate: string) => {
  const d = new Date(isoDate + 'T00:00:00');
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
};

const chartData = computed(() => ({
  labels: props.daily.map((d, i) => {
    if (i === 0) return formatAxisDate(d.date);
    if (i === props.daily.length - 1) return formatAxisDate(d.date);
    return '';
  }),
  datasets: [
    {
      data: props.daily.map((d) => d.cents / 100),
      backgroundColor: props.daily.map((_, i) => (i === peakIndex.value ? HIGHLIGHT_COLOR : NORMAL_COLOR)),
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
