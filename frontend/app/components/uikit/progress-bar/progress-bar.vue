<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <!-- This component might change based on Nuno's feedback -->
  <div
    :class="[
      `c-progress-bar c-progress-bar--${props.color} c-progress-bar--size-${props.size}`,
      { 'c-progress-bar--interactive': props.tooltips?.length },
    ]"
  >
    <template
      v-for="(value, index) in props.values"
      :key="`${value}-${index}`"
    >
      <lfx-tooltip
        v-if="props.tooltips?.[index]"
        class="c-progress-bar__value"
        :style="segmentStyle(value, index)"
        :content="props.tooltips[index]"
        placement="top"
      />
      <div
        v-else
        class="c-progress-bar__value"
        :style="segmentStyle(value, index)"
      />
    </template>
    <div
      v-if="props.label"
      class="c-progress-bar__label"
    >
      {{ props.label }}
    </div>
    <div
      v-if="!props.hideEmpty && totalFilled < 100"
      class="c-progress-bar__empty"
    />
  </div>
</template>

<script setup lang="ts">
import type { ProgressBarType } from './types/progress-bar.types';
import LfxTooltip from '~/components/uikit/tooltip/tooltip.vue';

const props = withDefaults(
  defineProps<{
    values: number[];
    colors?: string[];
    size?: 'small' | 'normal' | 'large';
    // TODO: change this once we have the correct types
    color?: ProgressBarType;
    label?: string;
    hideEmpty?: boolean;
    tooltips?: string[];
    minSegmentWidth?: number;
  }>(),
  {
    color: 'normal',
    size: 'normal',
    hideEmpty: false,
    label: undefined,
    colors: undefined,
    tooltips: undefined,
    minSegmentWidth: undefined,
  },
);

const segmentStyle = (value: number, index: number) => ({
  width: `${value}%`,
  // Keep tiny-but-nonzero segments visible without distorting the larger ones.
  ...(props.minSegmentWidth && value > 0 ? { minWidth: `${props.minSegmentWidth}px` } : {}),
  ...(props.colors?.[index] ? { backgroundColor: props.colors[index] } : {}),
});

const totalFilled = computed(() => props.values.reduce((sum, v) => sum + v, 0));
</script>

<script lang="ts">
export default {
  name: 'LfxProgressBar',
};
</script>
