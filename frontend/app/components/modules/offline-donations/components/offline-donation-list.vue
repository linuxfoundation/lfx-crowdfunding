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
  </lfx-card>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useQuery } from '@tanstack/vue-query';
import { formatCents } from '../config/offline-donation.config';
import LfxCard from '~/components/uikit/card/card.vue';
import LfxChip from '~/components/uikit/chip/chip.vue';
import LfxSpinner from '~/components/uikit/spinner/spinner.vue';
import type { OfflineDonationList } from '#shared/types/offline-donation.types';

const props = defineProps<{ initiativeId: string }>();

const { data, isLoading, isError } = useQuery<OfflineDonationList>({
  queryKey: ['offline-donations', computed(() => props.initiativeId)] as const,
  queryFn: () => $fetch<OfflineDonationList>('/api/offline-donations', { query: { initiativeId: props.initiativeId } }),
});

const rows = computed(() => data.value?.data ?? []);
</script>

<script lang="ts">
export default {
  name: 'OfflineDonationList',
};
</script>
