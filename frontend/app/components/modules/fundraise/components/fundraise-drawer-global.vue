<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <fundraise-drawer
    v-if="isOpen"
    v-model="isOpen"
  />
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia';
import { useFundraiseDrawerStore } from '../store/fundraise-drawer.store';
import FundraiseDrawer from './fundraise-drawer.vue';

const fundraiseDrawerStore = useFundraiseDrawerStore();
const { isOpen } = storeToRefs(fundraiseDrawerStore);
</script>

<script lang="ts">
export default {
  name: 'FundraiseDrawerGlobal',
};
</script>
