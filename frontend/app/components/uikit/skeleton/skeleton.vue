<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div
    class="animate-pulse bg-neutral-200 rounded-md"
    :class="[props.rounded ? 'rounded-full' : 'rounded-md', props.customClass]"
    :style="{ width: props.width, height: props.height }"
    aria-hidden="true"
  />
</template>

<script setup lang="ts">
const props = withDefaults(
  defineProps<{
    width?: string;
    height?: string;
    rounded?: boolean;
    customClass?: string;
  }>(),
  {
    width: '100%',
    height: '1rem',
    rounded: false,
    customClass: undefined,
  },
);
</script>

<script lang="ts">
export default {
  name: 'LfxSkeleton',
};
</script>
