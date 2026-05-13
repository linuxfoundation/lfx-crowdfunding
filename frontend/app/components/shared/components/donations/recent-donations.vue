<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="flex flex-col gap-6">
    <div class="flex items-center justify-between">
      <p class="text-base font-semibold text-neutral-900 leading-6">Recent donations</p>
      <a
        v-if="showSeeAllLink"
        href="#"
        class="text-sm text-accent-500 hover:text-accent-600 font-medium leading-4"
      >
        See all
      </a>
    </div>

    <div class="flex flex-col gap-4">
      <donations-row
        v-for="donation in donations"
        :key="donation.id"
        :donation="donation"
        :show-initiative-link="showInitiativeLink"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import DonationsRow from './donations-row.vue';
import type { RecentDonation } from '#shared/types/initiative-detail.types';

withDefaults(
  defineProps<{
    donations: RecentDonation[];
    showInitiativeLink?: boolean;
    showSeeAllLink?: boolean;
  }>(),
  {
    showSeeAllLink: true,
  },
);
</script>

<script lang="ts">
export default {
  name: 'RecentDonations',
};
</script>
