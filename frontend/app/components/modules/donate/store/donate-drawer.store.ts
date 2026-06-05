// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { defineStore } from 'pinia';
import { ref } from 'vue';
import type { FundDistributionItem } from '~/types/fundraise.types';

export interface DonateDrawerInitiative {
  id: string;
  name: string;
  logoUrl?: string;
  fundDistribution?: FundDistributionItem[];
}

export const useDonateDrawerStore = defineStore('donateDrawer', () => {
  const isOpen = ref(false);
  const initiative = ref<DonateDrawerInitiative | null>(null);

  const openDonateDrawer = (data: DonateDrawerInitiative) => {
    initiative.value = data;
    isOpen.value = true;
  };

  const closeDonateDrawer = () => {
    isOpen.value = false;
    initiative.value = null;
  };

  return { isOpen, initiative, openDonateDrawer, closeDonateDrawer };
});
