// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { defineStore } from 'pinia';
import { ref } from 'vue';
import type { ShareData } from '~/types/share.types';

export const useShareModalStore = defineStore('shareModal', () => {
  const isOpen = ref(false);
  const shareData = ref<ShareData | null>(null);

  const openShareModal = (data: ShareData) => {
    shareData.value = data;
    isOpen.value = true;
  };

  const closeShareModal = () => {
    isOpen.value = false;
    shareData.value = null;
  };

  return { isOpen, shareData, openShareModal, closeShareModal };
});
