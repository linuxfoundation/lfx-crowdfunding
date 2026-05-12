<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="flex items-center gap-3">
    <lfx-avatar
      :type="donation.donorType"
      size="normal"
      :src="donation.donorLogoUrl"
      class="shrink-0"
    />

    <div class="flex-1 min-w-0 flex flex-col">
      <div class="flex items-start justify-between gap-2">
        <span class="text-xs font-semibold text-neutral-900 leading-4 truncate">
          {{ donation.donorName }}
        </span>
        <span class="text-xs text-neutral-900 leading-4 shrink-0">
          {{ formatAmount(donation.amountCents) }}
        </span>
      </div>
      <div class="flex items-center text-[10px] leading-[14px]">
        <template v-if="showInitiativeLink && donation.initiativeId && donation.initiativeName">
          <NuxtLink
            :to="`/initiatives/${donation.initiativeId}`"
            class="text-accent-500 hover:text-accent-600 truncate shrink min-w-0"
          >
            {{ donation.initiativeName }}
          </NuxtLink>
          <span class="text-neutral-500 mx-0.5 shrink-0">・</span>
        </template>
        <span class="text-neutral-500 shrink-0">{{ donation.timeAgo }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { RecentDonation } from '#shared/types/initiative-detail.types';
import LfxAvatar from '~/components/uikit/avatar/avatar.vue';

defineProps<{
  donation: RecentDonation;
  showInitiativeLink?: boolean;
}>();

const formatAmount = (cents: number): string => {
  const dollars = cents / 100;
  if (dollars >= 1_000_000) return `$${(dollars / 1_000_000).toFixed(1).replace(/\.0$/, '')}M`;
  if (dollars >= 1_000) return `$${Math.round(dollars / 1_000)}K`;
  return `$${dollars.toLocaleString()}`;
};
</script>

<script lang="ts">
export default {
  name: 'DonationsRow',
};
</script>
