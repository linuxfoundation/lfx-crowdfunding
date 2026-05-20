<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <button
    type="button"
    class="c-icon-button"
    :class="[
      `c-icon-button--${props.size}`,
      `c-icon-button--${props.type}`,
      { 'is-disabled': props.disabled || props.loading },
    ]"
    :disabled="props.disabled || props.loading"
  >
    <lfx-icon
      v-if="props.loading"
      name="spinner-third"
      class="animate-spin"
      :size="props.iconSize"
    />
    <slot v-else>
      <lfx-icon
        :name="props.icon"
        :type="props.iconType"
        :size="props.iconSize"
      />
    </slot>
  </button>
</template>

<script setup lang="ts">
import LfxIcon from '~/components/uikit/icon/icon.vue';
import type { IconButtonSize, IconButtonType } from '~/components/uikit/icon-button/types/icon-button.types';
import type { IconType } from '~/components/uikit/icon/types/icon.types';

const props = withDefaults(
  defineProps<{
    icon: string;
    type?: IconButtonType;
    size?: IconButtonSize;
    iconType?: IconType;
    iconSize?: number;
    disabled?: boolean;
    loading?: boolean;
  }>(),
  {
    type: 'default',
    size: 'medium',
    iconType: 'light',
    iconSize: 16,
    disabled: false,
    loading: false,
  },
);
</script>

<script lang="ts">
export default {
  name: 'LfxIconButton',
};
</script>
