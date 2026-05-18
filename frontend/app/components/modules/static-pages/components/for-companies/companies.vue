<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="border-t border-neutral-200 pt-16 flex flex-col gap-10">
    <div class="flex flex-col gap-3">
      <h2 class="md:text-2xl text-xl font-semibold leading-9 text-neutral-900">Companies already investing</h2>
      <p class="md:text-base text-sm font-normal leading-6 text-neutral-900">
        Industry leaders funding the open source projects they depend on.
      </p>
    </div>

    <!-- Loading -->
    <div
      v-if="isLoading"
      class="border border-neutral-200 rounded-xl flex md:flex-row flex-col"
    >
      <div
        v-for="col in 4"
        :key="col"
        class="flex-1 flex flex-col"
        :class="{ 'md:border-l border-t md:border-t-0 border-neutral-200': col > 1 }"
      >
        <div
          v-for="row in 4"
          :key="row"
          class="flex gap-4 items-center p-6"
          :class="{ 'border-b border-neutral-200': row < 4 }"
        >
          <lfx-skeleton
            :rounded="true"
            width="2.5rem"
            height="2.5rem"
          />
          <div class="flex flex-col gap-1">
            <lfx-skeleton
              height="1.25rem"
              width="6rem"
            />
            <lfx-skeleton
              height="1rem"
              width="5rem"
            />
          </div>
        </div>
      </div>
    </div>

    <!-- Error -->
    <div
      v-else-if="error"
      class="flex items-center gap-2 text-negative-600"
    >
      <lfx-icon
        name="circle-exclamation"
        type="solid"
        :size="16"
      />
      <span class="text-sm leading-5">Failed to load companies.</span>
    </div>

    <!-- 3-column grid -->
    <div
      v-else
      class="border border-neutral-200 rounded-xl grid md:grid-cols-3 grid-cols-1"
    >
      <div
        v-for="(company, index) in companies"
        :key="company.id"
        class="flex gap-4 items-center p-6 min-w-0"
        :class="[
          // Mobile (1-col): bottom border between items
          index < companies.length - 1 ? 'border-b border-neutral-200' : '',
          // Remove that mobile bottom border on desktop where desktop logic says no bottom border
          index < companies.length - 1 && !(index < companies.length - (companies.length % 3 || 3))
            ? 'md:border-b-0'
            : '',
          // Desktop (3-col): right border between columns
          index % 3 < 2 ? 'md:border-r md:border-neutral-200' : '',
          // Desktop: bottom border between rows
          index < companies.length - (companies.length % 3 || 3) ? 'md:border-b md:border-neutral-200' : '',
        ]"
      >
        <lfx-avatar
          type="organization"
          :src="company.logoUrl"
          size="large"
          class="shrink-0"
        />
        <div class="flex flex-col min-w-0">
          <p class="text-sm font-semibold leading-5 text-neutral-900 truncate">{{ company.name }}</p>
          <p class="text-xs leading-4 text-neutral-500">
            {{ formatContributed(company.contributedCents) }} contributed
          </p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import LfxAvatar from '~/components/uikit/avatar/avatar.vue';
import LfxIcon from '~/components/uikit/icon/icon.vue';
import LfxSkeleton from '~/components/uikit/skeleton/skeleton.vue';
import { formatNumberCurrency } from '~/utils/formatter';
import type { InvestingCompany } from '#shared/types/static-pages.types';

defineProps<{
  companies: InvestingCompany[];
  isLoading: boolean;
  error: Error | null;
}>();

const formatContributed = (cents: number) => formatNumberCurrency(cents / 100, 'USD');
</script>

<script lang="ts">
export default {
  name: 'ForCompaniesCompanies',
};
</script>
