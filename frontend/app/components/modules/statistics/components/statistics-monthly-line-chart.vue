<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div
    class="flex-1 w-full flex flex-col min-w-0"
    :aria-label="
      props.buckets.length === 0
        ? 'Monthly donations line chart. No data available.'
        : `Monthly donations line chart. Peak month: ${peakLabel} with ${peakAmount}.`
    "
    role="img"
  >
    <Line
      :data="chartData"
      :options="chartOptions"
      class="w-full"
      style="height: 90px"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { Line } from 'vue-chartjs';
import {
  Chart as ChartJS,
  LineController,
  LineElement,
  PointElement,
  CategoryScale,
  LinearScale,
  Filler,
  Tooltip,
} from 'chart.js';
import type { ScriptableContext } from 'chart.js';
import { formatNumberCurrency } from '~/utils/formatter';
import type { MonthlyBucket } from '#shared/types/statistics.types';

ChartJS.register(LineController, LineElement, PointElement, CategoryScale, LinearScale, Filler, Tooltip);

const MONTH_ABBR = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];

const LINE_COLOR = '#009aff';

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

const chartData = computed(() => ({
  labels: props.buckets.map((b, i) => {
    if (i === 0 || i === props.buckets.length - 1) return MONTH_ABBR[b.month - 1];
    return '';
  }),
  datasets: [
    {
      data: props.buckets.map((b) => b.totalCents / 100),
      borderColor: LINE_COLOR,
      borderWidth: 2,
      fill: true,
      backgroundColor: (context: ScriptableContext<'line'>) => {
        const { ctx, chartArea } = context.chart;
        if (!chartArea) return 'rgba(0, 154, 255, 0.12)';
        const gradient = ctx.createLinearGradient(0, chartArea.top, 0, chartArea.bottom);
        gradient.addColorStop(0, 'rgba(0, 154, 255, 0.25)');
        gradient.addColorStop(1, 'rgba(0, 154, 255, 0)');
        return gradient;
      },
      tension: 0.4,
      pointRadius: 0,
      pointHoverRadius: 4,
      pointHoverBackgroundColor: LINE_COLOR,
      pointHoverBorderColor: '#ffffff',
      pointHoverBorderWidth: 2,
    },
  ],
}));

const chartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  interaction: { intersect: false, mode: 'index' as const },
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
      beginAtZero: true,
      grid: { display: false },
    },
  },
};
</script>

<script lang="ts">
export default {
  name: 'StatisticsMonthlyLineChart',
};
</script>
