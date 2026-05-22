<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <share-modal
    v-if="isOpen && shareData"
    v-model="isOpen"
    :defaults="shareData"
  />
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia';
import { useShareModalStore } from './store/share-modal.store';
import ShareModal from './share-modal.vue';

const shareModalStore = useShareModalStore();
const { isOpen, shareData } = storeToRefs(shareModalStore);
</script>

<script lang="ts">
export default {
  name: 'ShareModalGlobal',
};
</script>
