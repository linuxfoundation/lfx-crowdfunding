<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="flex flex-col gap-6">
    <div class="flex items-center justify-between">
      <p class="text-base font-semibold text-neutral-900 leading-6">Recent donations</p>
      <a
        v-if="showSeeAllLink && !isLoading && donations.length"
        href="#"
        class="text-sm text-accent-500 hover:text-accent-600 font-medium leading-4"
      >
        See all
      </a>
    </div>

    <!-- Loading skeleton -->
    <div
      v-if="isLoading"
      class="flex flex-col gap-4"
    >
      <div
        v-for="n in 5"
        :key="n"
        class="flex items-center gap-3"
      >
        <lfx-skeleton
          :rounded="true"
          width="2.5rem"
          height="2.5rem"
        />
        <div class="flex-1 flex flex-col gap-1">
          <lfx-skeleton
            height="0.875rem"
            width="70%"
          />
          <lfx-skeleton
            height="0.75rem"
            width="40%"
          />
        </div>
      </div>
    </div>

    <!-- Empty state -->
    <p
      v-else-if="!donations.length"
      class="text-sm text-neutral-500"
    >
      No donations yet.
    </p>

    <!-- Donation rows -->
    <div
      v-else
      class="flex flex-col gap-4"
    >
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
import LfxSkeleton from '~/components/uikit/skeleton/skeleton.vue';
import type { RecentDonation } from '#shared/types/initiative-detail.types';

withDefaults(
  defineProps<{
    donations: RecentDonation[];
    isLoading?: boolean;
    showInitiativeLink?: boolean;
    showSeeAllLink?: boolean;
  }>(),
  {
    isLoading: false,
    showSeeAllLink: true,
  },
);
</script>

<script lang="ts">
export default {
  name: 'RecentDonations',
};
</script>
