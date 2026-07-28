<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <button
    v-bind="$attrs"
    type="button"
    class="inline-flex items-center gap-1 text-xs font-medium leading-4 transition-opacity hover:opacity-70 disabled:opacity-40 disabled:cursor-not-allowed"
    :class="{
      'text-neutral-500': props.variant === 'neutral',
      'text-accent-500': props.variant === 'accent',
      'text-negative-500': props.variant === 'negative',
    }"
  >
    <lfx-icon
      v-if="props.icon && props.iconPosition === 'left'"
      :name="props.icon"
      :type="props.iconType"
      :size="12"
    />
    <slot>{{ props.label }}</slot>
    <lfx-icon
      v-if="props.icon && props.iconPosition === 'right'"
      :name="props.icon"
      :type="props.iconType"
      :size="12"
    />
  </button>
</template>

<script setup lang="ts">
import type { LinkButtonStyle } from './types/link-button.types';
import type { IconType } from '~/components/uikit/icon/types/icon.types';
import LfxIcon from '~/components/uikit/icon/icon.vue';

const props = withDefaults(
  defineProps<{
    label?: string;
    icon?: string;
    iconType?: IconType;
    iconPosition?: 'left' | 'right';
    variant?: LinkButtonStyle;
  }>(),
  {
    variant: 'neutral',
    iconType: 'light',
    iconPosition: 'left',
    label: undefined,
    icon: undefined,
  },
);
</script>

<script lang="ts">
export default {
  name: 'LfxLinkButton',
};
</script>
