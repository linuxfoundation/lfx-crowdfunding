<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <lfx-card class="p-6">
    <div class="flex gap-6 items-start md:flex-row flex-col md:gap-4">
      <div
        v-for="stat in stats"
        :key="stat.label"
        class="flex flex-1 min-w-0 flex-col gap-1"
      >
        <div class="flex items-center gap-2">
          <div
            class="size-6 rounded-full flex items-center justify-center shrink-0"
            :class="stat.iconBg"
          >
            <lfx-icon
              :name="stat.icon"
              type="solid"
              :size="12"
              class="text-white"
            />
          </div>
          <span class="text-sm text-neutral-900 leading-5 whitespace-nowrap">{{ stat.label }}</span>
        </div>
        <p class="text-4xl text-neutral-900 leading-[56px]">
          {{ stat.value }}
        </p>
      </div>
    </div>
  </lfx-card>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { FinancialSummary } from '#shared/types/initiative-detail.types';
import LfxCard from '~/components/uikit/card/card.vue';
import LfxIcon from '~/components/uikit/icon/icon.vue';
import { formatAmountCents } from '~/utils/currency';

const props = defineProps<{ summary: FinancialSummary }>();

const formatAmount = formatAmountCents;

const stats = computed(() => [
  {
    label: 'Total received',
    icon: 'arrow-trend-up',
    iconBg: 'bg-positive-500',
    value: formatAmount(props.summary.totalReceivedCents),
  },
  {
    label: 'Total expenses',
    icon: 'arrow-trend-down',
    iconBg: 'bg-warning-500',
    value: formatAmount(props.summary.totalExpensesCents),
  },
  {
    label: 'Balance',
    icon: 'sack-dollar',
    iconBg: 'bg-accent-500',
    value: formatAmount(props.summary.balanceCents),
  },
]);
</script>

<script lang="ts">
export default {
  name: 'InitiativeDetailFinancialSummary',
};
</script>
