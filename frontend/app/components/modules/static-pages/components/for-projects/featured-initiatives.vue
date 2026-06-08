<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="pt-16 flex flex-col gap-10">
    <div class="flex flex-col gap-3">
      <h2 class="text-2xl font-semibold leading-9 text-neutral-900">Initiatives raising with Crowdfunding</h2>
      <p class="text-base font-normal leading-6 text-neutral-900">
        Join the initiatives that trust LFX Crowdfunding to fund their most important work.
      </p>
    </div>

    <div class="border border-neutral-200 rounded-xl flex flex-col">
      <!-- Loading state -->
      <template v-if="isLoading">
        <div
          v-for="n in 6"
          :key="n"
          class="flex gap-8 items-center p-6"
          :class="{ 'border-b border-neutral-200': n < 6 }"
        >
          <div class="flex-1 flex items-center gap-3">
            <lfx-skeleton
              :rounded="true"
              width="2.5rem"
              height="2.5rem"
            />
            <lfx-skeleton
              height="1.25rem"
              width="10rem"
            />
          </div>
          <div class="flex-1 flex flex-col gap-2">
            <lfx-skeleton
              height="1rem"
              class="w-full"
            />
            <lfx-skeleton
              height="0.25rem"
              class="w-full"
            />
          </div>
          <lfx-skeleton
            width="1.25rem"
            height="1.25rem"
          />
        </div>
      </template>

      <!-- Error state -->
      <div
        v-else-if="error"
        class="flex items-center gap-2 text-negative-600 p-6"
      >
        <lfx-icon
          name="circle-exclamation"
          type="solid"
          :size="16"
        />
        <span class="text-sm leading-5">Failed to load featured initiatives.</span>
      </div>

      <!-- Data rows -->
      <template v-else>
        <NuxtLink
          v-for="(initiative, index) in initiatives"
          :key="initiative.id"
          :to="`/initiatives/${initiative.slug}`"
          class="flex gap-8 md:items-center items-start p-6 hover:bg-neutral-50 transition-colors md:flex-row flex-col"
          :class="{ 'border-b border-neutral-200': index < initiatives.length - 1 }"
        >
          <!-- Left: logo + name -->
          <div class="flex-1 w-full flex items-center gap-3 min-w-0">
            <lfx-avatar
              type="organization"
              :src="initiative.logoUrl"
              size="normal"
              class="shrink-0"
            />
            <p class="text-sm font-semibold leading-5 text-neutral-900 truncate">
              {{ initiative.name }}
            </p>
          </div>

          <!-- Middle: amounts + progress bar -->
          <div class="flex-1 w-full flex flex-col gap-2 min-w-0">
            <div class="flex gap-4 items-start">
              <span class="flex-1 text-xs font-semibold leading-4 text-neutral-900">
                {{ formatRaised(initiative.raisedCents) }} fundraised
              </span>
              <span class="flex-1 text-xs font-normal leading-4 text-neutral-500 text-right">
                {{ initiative.supporterCount }} supporters
              </span>
            </div>
            <lfx-progress-bar
              :values="[progressPercent(initiative)]"
              size="small"
            />
          </div>

          <!-- Right: chevron -->
          <lfx-icon
            name="angle-right"
            type="light"
            :size="16"
            class="text-neutral-400 shrink-0 md:block hidden md:relative absolute"
          />
        </NuxtLink>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import LfxAvatar from '~/components/uikit/avatar/avatar.vue';
import LfxIcon from '~/components/uikit/icon/icon.vue';
import LfxSkeleton from '~/components/uikit/skeleton/skeleton.vue';
import LfxProgressBar from '~/components/uikit/progress-bar/progress-bar.vue';
import { formatNumberCurrency } from '~/utils/formatter';
import type { FeaturedInitiative } from '#shared/types/static-pages.types';

defineProps<{
  initiatives: FeaturedInitiative[];
  isLoading: boolean;
  error: Error | null;
}>();

const formatRaised = (cents: number) => formatNumberCurrency(cents / 100, 'USD');

const progressPercent = (initiative: FeaturedInitiative): number => {
  if (initiative.goalCents <= 0) return 0;
  return Math.min(100, (initiative.raisedCents / initiative.goalCents) * 100);
};
</script>

<script lang="ts">
export default {
  name: 'ForProjectsFeaturedInitiatives',
};
</script>
