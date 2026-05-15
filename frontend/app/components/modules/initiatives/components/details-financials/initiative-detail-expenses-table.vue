<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <lfx-card class="p-6 flex flex-col gap-6">
    <p class="text-base font-semibold text-neutral-900 leading-6">Expense breakdown</p>

    <table class="w-full">
      <thead>
        <tr>
          <th class="text-xs font-medium text-neutral-500 text-left py-2 w-[140px] md:visible hidden">Date</th>
          <th class="text-xs font-medium text-neutral-500 text-left py-2 px-3 w-[140px] md:visible hidden">Category</th>
          <th class="text-xs font-medium text-neutral-500 text-left py-2 px-3">Description</th>
          <th class="text-xs font-medium text-neutral-500 text-right py-2 w-[140px]">Amount</th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="record in expenses"
          :key="record.id"
          class="border-t border-neutral-200"
        >
          <td class="text-xs text-neutral-900 py-4 w-[140px] md:visible hidden">{{ record.date }}</td>
          <td class="py-4 px-3 w-[140px] md:visible hidden">
            <lfx-tag
              variation="neutral"
              size="small"
              >{{ record.category }}</lfx-tag
            >
          </td>
          <td class="text-xs text-neutral-900 py-4 px-3 flex flex-col">
            {{ record.description }}
            <span class="text-xs text-neutral-500 font-normal">{{ record.category }}</span>
          </td>
          <td class="text-xs font-semibold text-neutral-900 text-right py-4 w-[140px]">
            <div class="flex flex-col">
              {{ formatAmount(record.amountCents) }}
              <span class="text-xs text-neutral-500 font-normal">{{ record.date }}</span>
            </div>
          </td>
        </tr>
      </tbody>
    </table>
  </lfx-card>
</template>

<script setup lang="ts">
import type { ExpenseRecord } from '#shared/types/initiative-detail.types';
import LfxCard from '~/components/uikit/card/card.vue';
import LfxTag from '~/components/uikit/tag/tag.vue';

defineProps<{ expenses: ExpenseRecord[] }>();

const formatAmount = (cents: number): string => {
  const dollars = cents / 100;
  if (dollars >= 1_000_000) return `$${(dollars / 1_000_000).toFixed(1).replace(/\.0$/, '')}M`;
  if (dollars >= 1_000) return `$${Math.round(dollars / 1_000)}K`;
  return `$${dollars.toLocaleString()}`;
};
</script>

<script lang="ts">
export default {
  name: 'InitiativeDetailExpensesTable',
};
</script>
