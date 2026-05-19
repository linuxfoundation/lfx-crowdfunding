<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <lfx-button
    :label="label"
    icon="box-dollar"
    button-style="pill"
    :type="type"
    v-bind="$attrs"
    @click="openFundraiseDrawer()"
  />
</template>

<script setup lang="ts">
import LfxButton from '~/components/uikit/button/button.vue';
import { useFundraiseDrawerStore } from '~/components/modules/fundraise/store/fundraise-drawer.store';

withDefaults(
  defineProps<{
    type?: 'primary' | 'secondary' | 'tertiary' | 'transparent' | 'ghost' | 'outline' | 'nav';
    label?: string;
  }>(),
  { type: 'transparent', label: 'Start Fundraise' },
);

const { openFundraiseDrawer } = useFundraiseDrawerStore();
</script>

<script lang="ts">
export default {
  name: 'StartFundraiseButton',
  inheritAttrs: false,
};
</script>
