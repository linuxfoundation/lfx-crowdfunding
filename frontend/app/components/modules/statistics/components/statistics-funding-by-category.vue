<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <lfx-card class="p-6 flex flex-col gap-6">
    <p class="text-base font-semibold text-neutral-900 leading-6">Funding by category</p>

    <template v-if="isLoading">
      <lfx-skeleton
        height="0.375rem"
        class="w-full"
      />
      <div class="flex flex-col gap-3">
        <div
          v-for="n in 5"
          :key="n"
          class="flex items-center justify-between"
        >
          <div class="flex items-center gap-3">
            <lfx-skeleton
              :rounded="true"
              width="1.5rem"
              height="1.5rem"
            />
            <lfx-skeleton
              height="0.875rem"
              width="6rem"
            />
          </div>
          <lfx-skeleton
            height="0.875rem"
            width="8rem"
          />
        </div>
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
      <span class="text-sm">Failed to load categories.</span>
    </div>

    <template v-else-if="categories.length">
      <!-- Combined stacked bar across all categories -->
      <lfx-progress-bar
        :values="categories.map(categoryPercent)"
        :colors="categories.map((c) => CATEGORY_HEX[c.id] ?? '#e2e8f0')"
        :hide-empty="true"
      />

      <!-- Category rows -->
      <div class="flex flex-col gap-3">
        <div
          v-for="category in categories"
          :key="category.id"
          class="flex items-center justify-between"
        >
          <!-- Icon bubble + name -->
          <div class="flex items-center gap-3">
            <div
              class="size-6 rounded-full flex items-center justify-center shrink-0"
              :style="categoryBg(category.id)"
            >
              <lfx-icon
                :name="category.icon"
                type="solid"
                :size="12"
                class="text-white"
              />
            </div>
            <span class="text-sm font-semibold text-neutral-900">{{ category.name }}</span>
          </div>

          <!-- Percent + donated -->
          <span class="text-sm text-neutral-600 whitespace-nowrap">
            {{ categoryPercent(category) }}% ・ {{ formatShort(category.raisedCents) }} donated
          </span>
        </div>
      </div>
    </template>
  </lfx-card>
</template>

<script setup lang="ts">
import LfxCard from '~/components/uikit/card/card.vue';
import LfxIcon from '~/components/uikit/icon/icon.vue';
import LfxSkeleton from '~/components/uikit/skeleton/skeleton.vue';
import LfxProgressBar from '~/components/uikit/progress-bar/progress-bar.vue';
import { formatNumberShort } from '~/utils/formatter';
import { CATEGORY_HEX } from '~/config/statistics/categories';
import type { FundingCategory } from '#shared/types/statistics.types';

const props = defineProps<{
  categories: FundingCategory[];
  isLoading: boolean;
  error: Error | null;
}>();

const categoryBg = (id: string) => {
  const hex = CATEGORY_HEX[id as keyof typeof CATEGORY_HEX];
  return hex ? `background-color: ${hex}` : 'background-color: #e2e8f0';
};

const total = () => props.categories.reduce((sum, c) => sum + c.raisedCents, 0);

const categoryPercent = (c: FundingCategory): number => {
  const t = total();
  return t > 0 ? Math.round((c.raisedCents / t) * 100) : 0;
};

const formatShort = (cents: number) => formatNumberShort(cents / 100);
</script>

<script lang="ts">
export default {
  name: 'StatisticsFundingByCategory',
};
</script>
