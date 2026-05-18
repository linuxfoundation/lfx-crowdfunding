<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <donate-drawer
    v-if="isOpen && initiative"
    v-model="isOpen"
    :initiative="initiative"
  />
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia';
import { useDonateDrawerStore } from '../store/donate-drawer.store';
import DonateDrawer from './donate-drawer.vue';

const donateDrawerStore = useDonateDrawerStore();
const { isOpen, initiative } = storeToRefs(donateDrawerStore);
</script>

<script lang="ts">
export default {
  name: 'DonateDrawerGlobal',
};
</script>
