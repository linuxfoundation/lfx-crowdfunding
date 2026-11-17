<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <lfx-card class="p-6 flex flex-col gap-4">
    <p class="text-sm font-semibold text-neutral-900">Recorded offline donations</p>

    <div
      v-if="isLoading"
      class="flex justify-center py-4"
    >
      <lfx-spinner />
    </div>
    <p
      v-else-if="isError"
      class="text-sm text-neutral-600"
    >
      Recorded donations couldn't be loaded.
    </p>
    <p
      v-else-if="!rows.length"
      class="text-sm text-neutral-600"
    >
      None recorded for this initiative yet.
    </p>
    <ul
      v-else
      class="flex flex-col divide-y divide-neutral-100"
    >
      <li
        v-for="row in rows"
        :key="row.id"
        class="flex items-start gap-4 py-3"
      >
        <div class="flex-1 min-w-0">
          <p class="text-sm font-medium text-neutral-900 truncate">
            {{ row.organizationName }}
            <span class="text-xs font-normal text-neutral-500">{{ row.organizationDomain }}</span>
          </p>
          <p class="text-xs text-neutral-600 truncate">
            {{ row.receivedOn }} · {{ row.category }} · {{ row.sourceTxnId }}
          </p>
          <p
            v-if="row.recordedBy"
            class="text-xs text-neutral-500"
          >
            Recorded by {{ row.recordedBy }}
          </p>
        </div>
        <div class="flex flex-col items-end gap-1 shrink-0">
          <p class="text-sm font-semibold text-neutral-900">{{ formatCents(row.amountCents) }}</p>
          <lfx-chip
            v-if="row.status !== 'succeeded'"
            size="xsmall"
          >
            Not in the Ledger yet
          </lfx-chip>
        </div>
      </li>
    </ul>

    <div
      v-if="hasNextPage"
      class="flex justify-center"
    >
      <lfx-button
        :label="`Load more (${rows.length} of ${total})`"
        type="outline"
        size="small"
        button-style="pill"
        :loading="isFetchingNextPage"
        @click="fetchNextPage()"
      />
    </div>
  </lfx-card>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useInfiniteQuery } from '@tanstack/vue-query';
import {
  OFFLINE_DONATIONS_PAGE_SIZE,
  formatCents,
  nextOfflineDonationsOffset,
} from '../config/offline-donation.config';
import LfxButton from '~/components/uikit/button/button.vue';
import LfxCard from '~/components/uikit/card/card.vue';
import LfxChip from '~/components/uikit/chip/chip.vue';
import LfxSpinner from '~/components/uikit/spinner/spinner.vue';
import type { OfflineDonationList } from '#shared/types/offline-donation.types';

const props = defineProps<{ initiativeId: string }>();

// Paged so every recorded donation stays reachable, newest received first.
const { data, isLoading, isError, hasNextPage, isFetchingNextPage, fetchNextPage } = useInfiniteQuery({
  queryKey: ['offline-donations', computed(() => props.initiativeId)] as const,
  queryFn: ({ pageParam }) =>
    $fetch<OfflineDonationList>('/api/offline-donations', {
      query: { initiativeId: props.initiativeId, limit: OFFLINE_DONATIONS_PAGE_SIZE, offset: pageParam },
    }),
  initialPageParam: 0,
  getNextPageParam: (lastPage, allPages) => nextOfflineDonationsOffset(allPages, lastPage.total),
});

const rows = computed(() => data.value?.pages.flatMap((p) => p.data) ?? []);
const total = computed(() => data.value?.pages.at(-1)?.total ?? 0);
</script>

<script lang="ts">
export default {
  name: 'OfflineDonationList',
};
</script>
