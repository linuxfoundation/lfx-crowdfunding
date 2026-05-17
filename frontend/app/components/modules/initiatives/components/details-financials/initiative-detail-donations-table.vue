<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <lfx-card class="p-6 flex flex-col gap-6">
    <p class="text-base font-semibold text-neutral-900 leading-6">Donations received</p>

    <!-- Loading skeleton -->
    <div
      v-if="isLoading"
      class="flex flex-col gap-4"
    >
      <div
        v-for="n in 5"
        :key="n"
        class="flex items-center gap-3 border-t border-neutral-200 pt-4"
      >
        <lfx-skeleton
          :rounded="true"
          width="1.5rem"
          height="1.5rem"
        />
        <lfx-skeleton
          height="0.875rem"
          width="60%"
        />
        <lfx-skeleton
          height="0.875rem"
          width="15%"
          class="ml-auto"
        />
      </div>
    </div>

    <!-- Empty state -->
    <p
      v-else-if="!donations.length"
      class="text-sm text-neutral-500"
    >
      No donations received yet.
    </p>

    <!-- Table -->
    <template v-else>
      <table class="w-full">
        <thead>
          <tr>
            <th class="text-xs font-medium text-neutral-500 text-left py-2 w-[140px] md:visible hidden">Date</th>
            <th class="text-xs font-medium text-neutral-500 text-left py-2 md:px-3 pr-3">Supporter</th>
            <th class="text-xs font-medium text-neutral-500 text-left py-2 px-3 w-[140px] md:visible hidden">Type</th>
            <th class="text-xs font-medium text-neutral-500 text-right py-2 w-[140px]">Amount</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="record in donations"
            :key="record.id"
            class="border-t border-neutral-200"
          >
            <td class="text-xs text-neutral-900 py-4 w-[140px] md:visible hidden">{{ record.date }}</td>
            <td class="py-4 md:px-3 pr-3">
              <div class="flex items-center gap-2">
                <lfx-avatar
                  :type="record.supporterType"
                  size="small"
                  :src="record.supporterLogoUrl"
                  class="shrink-0"
                />
                <span class="text-xs text-neutral-900 truncate">{{ record.supporterName }}</span>
              </div>
            </td>
            <td class="py-4 px-3 w-[140px] md:visible hidden">
              <lfx-tag
                variation="neutral"
                size="small"
                >{{ record.donorCategory }}</lfx-tag
              >
            </td>
            <td class="text-xs font-semibold text-neutral-900 text-right py-4 w-[140px] flex flex-col">
              {{ formatAmount(record.amountCents) }}
              <span class="text-xs text-neutral-500 font-normal">{{ record.date }}</span>
            </td>
          </tr>
        </tbody>
      </table>

      <div class="flex justify-center">
        <lfx-button
          label="Load more"
          type="outline"
          size="small"
          button-style="pill"
        />
      </div>
    </template>
  </lfx-card>
</template>

<script setup lang="ts">
import type { DonationRecord } from '#shared/types/initiative-detail.types';
import LfxCard from '~/components/uikit/card/card.vue';
import LfxAvatar from '~/components/uikit/avatar/avatar.vue';
import LfxTag from '~/components/uikit/tag/tag.vue';
import LfxButton from '~/components/uikit/button/button.vue';
import LfxSkeleton from '~/components/uikit/skeleton/skeleton.vue';

defineProps<{ donations: DonationRecord[]; isLoading?: boolean }>();

const formatAmount = (cents: number): string => {
  const dollars = cents / 100;
  if (dollars >= 1_000_000) return `$${(dollars / 1_000_000).toFixed(1).replace(/\.0$/, '')}M`;
  if (dollars >= 1_000) return `$${Math.round(dollars / 1_000)}K`;
  return `$${dollars.toLocaleString()}`;
};
</script>

<script lang="ts">
export default {
  name: 'InitiativeDetailDonationsTable',
};
</script>
