<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <lfx-card class="p-6 flex flex-col gap-6">
    <p class="text-base font-semibold text-neutral-900 leading-6">{{ title }}</p>

    <template v-if="isLoading">
      <div
        v-for="n in 5"
        :key="n"
        class="flex items-center gap-3"
      >
        <lfx-skeleton
          :rounded="true"
          width="2rem"
          height="2rem"
        />
        <lfx-skeleton
          height="1rem"
          width="60%"
        />
        <lfx-skeleton
          height="1rem"
          width="3rem"
          class="ml-auto"
        />
      </div>
    </template>

    <div
      v-else-if="error"
      class="flex items-center gap-2 text-negative-600"
    >
      <lfx-icon
        name="circle-exclamation"
        type="solid"
        :size="16"
      />
      <span class="text-sm">Failed to load top donors.</span>
    </div>

    <div
      v-else
      class="flex flex-col divide-y divide-neutral-100"
    >
      <div
        v-for="entry in entries"
        :key="entry.id"
        class="flex items-center gap-3 py-3 first:pt-0 last:pb-0"
      >
        <span class="w-7 text-sm text-neutral-400 text-center shrink-0">#{{ entry.rank }}</span>
        <lfx-avatar
          :type="entryType"
          :src="entry.logoUrl"
          size="small"
          class="shrink-0"
        />
        <span class="flex-1 min-w-0 text-sm font-medium text-neutral-900 truncate">{{ entry.name }}</span>
        <span class="text-sm text-neutral-900 shrink-0">{{ formatCurrency(entry.amountCents) }}</span>
      </div>
    </div>
  </lfx-card>
</template>

<script setup lang="ts">
import LfxCard from '~/components/uikit/card/card.vue';
import LfxIcon from '~/components/uikit/icon/icon.vue';
import LfxSkeleton from '~/components/uikit/skeleton/skeleton.vue';
import LfxAvatar from '~/components/uikit/avatar/avatar.vue';
import { formatNumberCurrency } from '~/utils/formatter';
import type { TopDonor } from '#shared/types/statistics.types';

defineProps<{
  title: string;
  entries: TopDonor[];
  entryType: 'organization' | 'member';
  isLoading: boolean;
  error: Error | null;
}>();

const formatCurrency = (cents: number) => formatNumberCurrency(cents / 100, 'USD');
</script>

<script lang="ts">
export default {
  name: 'StatisticsTopList',
};
</script>
