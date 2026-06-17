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
      class="border border-neutral-200 rounded-xl overflow-hidden grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-px bg-neutral-200"
    >
      <div
        v-for="n in 16"
        :key="n"
        class="flex gap-4 items-center p-6 bg-white"
      >
        <lfx-skeleton
          :rounded="true"
          width="2rem"
          height="2rem"
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

    <!-- Responsive grid: 1 / 2 / 4 columns (16 companies divide evenly at each breakpoint) -->
    <div
      v-else
      class="border border-neutral-200 rounded-xl overflow-hidden grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-px bg-neutral-200"
    >
      <div
        v-for="company in companies"
        :key="company.id"
        class="flex gap-4 items-center p-6 min-w-0 bg-white"
      >
        <lfx-avatar
          type="organization"
          :src="company.logoUrl"
          size="normal"
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
