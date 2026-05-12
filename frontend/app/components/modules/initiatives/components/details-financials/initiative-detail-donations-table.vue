<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <lfx-card class="p-6 flex flex-col gap-6">
    <p class="text-base font-semibold text-neutral-900 leading-6">Donations received</p>

    <table class="w-full">
      <thead>
        <tr>
          <th class="text-xs font-medium text-neutral-500 text-left py-2 w-[140px]">Date</th>
          <th class="text-xs font-medium text-neutral-500 text-left py-2 px-3">Supporter</th>
          <th class="text-xs font-medium text-neutral-500 text-left py-2 px-3 w-[140px]">Type</th>
          <th class="text-xs font-medium text-neutral-500 text-right py-2 w-[140px]">Amount</th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="record in donations"
          :key="record.id"
          class="border-t border-neutral-200"
        >
          <td class="text-xs text-neutral-900 py-4 w-[140px]">{{ record.date }}</td>
          <td class="py-4 px-3">
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
          <td class="py-4 px-3 w-[140px]">
            <lfx-tag
              variation="neutral"
              size="small"
              >{{ record.donorCategory }}</lfx-tag
            >
          </td>
          <td class="text-xs font-semibold text-neutral-900 text-right py-4 w-[140px]">
            {{ formatAmount(record.amountCents) }}
          </td>
        </tr>
      </tbody>
    </table>

    <div class="flex justify-center">
      <lfx-button
        label="Load more"
        type="outline"
        size="small"
      />
    </div>
  </lfx-card>
</template>

<script setup lang="ts">
import type { DonationRecord } from '#shared/types/initiative-detail.types';
import LfxCard from '~/components/uikit/card/card.vue';
import LfxAvatar from '~/components/uikit/avatar/avatar.vue';
import LfxTag from '~/components/uikit/tag/tag.vue';
import LfxButton from '~/components/uikit/button/button.vue';

defineProps<{ donations: DonationRecord[] }>();

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
