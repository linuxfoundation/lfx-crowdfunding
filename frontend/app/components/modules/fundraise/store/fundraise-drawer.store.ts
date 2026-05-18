// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { defineStore } from 'pinia';
import { ref } from 'vue';

export const useFundraiseDrawerStore = defineStore('fundraiseDrawer', () => {
  const isOpen = ref(false);

  const openFundraiseDrawer = () => {
    isOpen.value = true;
  };

  const closeFundraiseDrawer = () => {
    isOpen.value = false;
  };

  return { isOpen, openFundraiseDrawer, closeFundraiseDrawer };
});
