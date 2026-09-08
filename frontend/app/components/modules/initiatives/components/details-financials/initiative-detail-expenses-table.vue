<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <lfx-card class="p-6 flex flex-col gap-6">
    <p class="text-base font-semibold text-neutral-900 leading-6">Expense breakdown</p>

    <!-- Loading skeleton -->
    <initiative-detail-table-skeleton
      v-if="isLoading"
      :columns="[{ width: '15%' }, { width: '55%', class: 'ml-3' }, { width: '15%', class: 'ml-auto' }]"
    />

    <template v-else>
      <table class="w-full">
        <thead>
          <tr>
            <th class="text-xs font-medium text-neutral-500 text-left py-2 w-[140px] md:visible hidden">Date</th>
            <th class="text-xs font-medium text-neutral-500 text-left py-2 px-3 w-[140px] md:visible hidden">
              Category
            </th>
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

      <div
        v-if="hasMore"
        class="flex justify-center"
      >
        <lfx-button
          label="Load more"
          type="outline"
          size="small"
          button-style="pill"
          :loading="isLoadingMore"
          @click="$emit('load-more')"
        />
      </div>
    </template>
  </lfx-card>
</template>

<script setup lang="ts">
import InitiativeDetailTableSkeleton from './initiative-detail-table-skeleton.vue';
import type { ExpenseRecord } from '#shared/types/initiative-detail.types';
import LfxCard from '~/components/uikit/card/card.vue';
import LfxTag from '~/components/uikit/tag/tag.vue';
import LfxButton from '~/components/uikit/button/button.vue';
import { formatAmountCents } from '~/utils/currency';

defineProps<{ expenses: ExpenseRecord[]; isLoading?: boolean; hasMore?: boolean; isLoadingMore?: boolean }>();

defineEmits<{
  'load-more': [];
}>();

const formatAmount = formatAmountCents;
</script>

<script lang="ts">
export default {
  name: 'InitiativeDetailExpensesTable',
};
</script>
