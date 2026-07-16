<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div
    class="relative shrink-0"
    :style="{ width: `${size}px`, height: `${size}px` }"
  >
    <svg
      :width="size"
      :height="size"
      :viewBox="`0 0 ${size} ${size}`"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <defs>
        <linearGradient
          v-for="seg in gradientSegments"
          :id="seg.id"
          :key="seg.id"
          gradientUnits="userSpaceOnUse"
          :x1="size"
          y1="0"
          x2="0"
          :y2="size"
        >
          <stop
            offset="0%"
            :stop-color="(seg.color as DonutChartColor).from"
          />
          <stop
            offset="100%"
            :stop-color="(seg.color as DonutChartColor).to"
          />
        </linearGradient>
      </defs>

      <!-- Background track -->
      <circle
        :cx="center"
        :cy="center"
        :r="radius"
        :stroke="trackColor"
        :stroke-width="strokeWidth"
        fill="none"
      />

      <!-- Segments -->
      <circle
        v-for="(seg, i) in resolvedSegments"
        :key="i"
        :cx="center"
        :cy="center"
        :r="radius"
        :stroke="seg.stroke"
        :stroke-width="strokeWidth"
        stroke-linecap="round"
        fill="none"
        :stroke-dasharray="`${seg.dash} ${circumference}`"
        :transform="`rotate(${seg.startAngle} ${center} ${center})`"
      />
    </svg>

    <!-- Center slot -->
    <div class="absolute inset-0 flex items-center justify-center">
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, useId } from 'vue';
import { isGradient } from './types/donut-chart.types';
import type {
  DonutChartColorProp,
  DonutChartColor,
  DonutChartSegment,
  GradientMeta,
  ResolvedSegment,
} from './types/donut-chart.types';

const props = withDefaults(
  defineProps<{
    /** 0–100 percentage value (single-segment mode) */
    value?: number;
    /** Diameter of the chart in px */
    size?: number;
    /** Ring thickness in px */
    strokeWidth?: number;
    /** Solid color string OR { from, to } gradient object (single-segment mode) */
    color?: DonutChartColorProp;
    /** Track (background ring) color */
    trackColor?: string;
    /** Multi-segment mode — each segment has its own value + color */
    segments?: DonutChartSegment[];
  }>(),
  {
    value: 0,
    size: 280,
    strokeWidth: 16,
    color: '#009AFF',
    trackColor: '#E2E8F0',
    segments: undefined,
  },
);

const center = computed(() => props.size / 2);
const radius = computed(() => center.value - props.strokeWidth);
const circumference = computed(() => 2 * Math.PI * radius.value);

// useId() is SSR/client-deterministic (unlike Math.random()), so gradient
// ids match between server-rendered and hydrated markup.
const instanceKey = useId();
const gradientId = (i: number) => `donut-gradient-${instanceKey}-${i}`;

// Normalize both modes into a unified segment list
const normalizedSegments = computed<DonutChartSegment[]>(() => {
  if (props.segments && props.segments.length > 0) {
    return props.segments;
  }
  return [{ value: Math.min(100, Math.max(0, props.value)), color: props.color }];
});

// Segments with gradient references that need <linearGradient> defs
const gradientSegments = computed<GradientMeta[]>(() =>
  normalizedSegments.value
    .map((seg, i) => ({ id: gradientId(i), color: seg.color }))
    .filter(({ color }) => isGradient(color)),
);

// Fully resolved segments — precompute dash length, rotation angle, and stroke reference
const resolvedSegments = computed<ResolvedSegment[]>(() => {
  let cumulativePct = 0;
  return normalizedSegments.value.map((seg, i) => {
    const pct = Math.min(100, Math.max(0, seg.value));
    const startAngle = -90 + (cumulativePct / 100) * 360;
    const dash = circumference.value * (pct / 100);
    const stroke = isGradient(seg.color) ? `url(#${gradientId(i)})` : (seg.color as string);
    cumulativePct += pct;
    return { dash, startAngle, stroke };
  });
});
</script>

<script lang="ts">
export default {
  name: 'LfxDonutChart',
};
</script>
