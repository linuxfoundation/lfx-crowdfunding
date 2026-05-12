<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="flex flex-col gap-6">
    <div class="flex items-center justify-between">
      <p class="text-base font-semibold text-neutral-900 leading-6">Recent donations</p>
      <a
        href="#"
        class="text-sm text-accent-500 hover:text-accent-600 font-medium leading-4"
      >
        View all
      </a>
    </div>

    <div class="flex flex-col gap-6">
      <donations-row
        v-for="donation in donations"
        :key="donation.id"
        :donation="donation"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import type { RecentDonation } from '#shared/types/initiative-detail.types';
import DonationsRow from '~/components/shared/components/donations/donations-row.vue';

defineProps<{ donations: RecentDonation[] }>();
</script>

<script lang="ts">
export default {
  name: 'InitiativeDetailRecentDonations',
};
</script>
