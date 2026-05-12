<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <button
    v-bind="$attrs"
    class="p-button"
    :class="[`p-button-${props.type}`, `p-button-${props.size}`, props.buttonStyle === 'pill' && 'p-button-pill']"
    :disabled="props.disabled || props.loading"
    type="button"
  >
    <lfx-icon
      v-if="props.loading"
      name="spinner-third"
      class="animate-spin"
      :size="14"
    />
    <template v-else>
      <lfx-icon
        v-if="props.icon && props.iconPosition === 'left'"
        :name="props.icon"
        :size="14"
      />
      <slot>{{ props.label }}</slot>
      <lfx-icon
        v-if="props.icon && props.iconPosition === 'right'"
        :name="props.icon"
        :size="14"
      />
    </template>
  </button>
</template>

<script setup lang="ts">
import type { ButtonStyle, ButtonType, ButtonSize, IconPosition } from './types/button.types';
import LfxIcon from '~/components/uikit/icon/icon.vue';

const props = withDefaults(
  defineProps<{
    label?: string;
    icon?: string;
    type?: ButtonType;
    buttonStyle?: ButtonStyle;
    loading?: boolean;
    size?: ButtonSize;
    iconPosition?: IconPosition;
    disabled?: boolean;
  }>(),
  {
    type: 'primary',
    size: 'medium',
    iconPosition: 'left',
    disabled: false,
    loading: false,
    label: undefined,
    icon: undefined,
    buttonStyle: 'rounded',
  },
);
</script>

<script lang="ts">
export default {
  name: 'LfxButton',
};
</script>
