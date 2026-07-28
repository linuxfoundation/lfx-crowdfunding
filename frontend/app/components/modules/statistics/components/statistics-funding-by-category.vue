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

    <template v-else-if="displayCategories.length">
      <!-- Combined stacked bar; "Others" sits last (right-most) -->
      <lfx-progress-bar
        :values="displayCategories.map(categoryBarPercent)"
        :colors="displayCategories.map((c) => categoryColor(c.name))"
        :tooltips="displayCategories.map(categoryTooltip)"
        :min-segment-width="6"
        :hide-empty="true"
      />

      <!-- Category rows -->
      <div class="flex flex-col gap-3">
        <template
          v-for="category in displayCategories"
          :key="category.id"
        >
          <!-- Divider between the main categories and the combined "Others" bucket -->
          <div
            v-if="category.id === OTHERS_ID"
            class="border-t border-neutral-200"
            aria-hidden="true"
          />

          <div class="flex items-center justify-between">
            <!-- Icon bubble + name -->
            <div class="flex items-center gap-3">
              <div
                class="size-6 rounded-full flex items-center justify-center shrink-0"
                :style="categoryBg(category.name)"
              >
                <lfx-icon
                  :name="getCategoryVisual(category.name).icon"
                  type="solid"
                  :size="12"
                  :class="getCategoryVisual(category.name).iconColor ?? 'text-white'"
                />
              </div>
              <span class="md:text-sm text-xs font-semibold text-neutral-900">{{
                getCategoryLabel(category.name)
              }}</span>
            </div>

            <!-- Percent + donated -->
            <span class="md:text-sm text-xs text-neutral-600 whitespace-nowrap">
              {{ categoryPercent(category) }}% ・ {{ formatShort(category.raisedCents) }} donated
            </span>
          </div>
        </template>
      </div>
    </template>
  </lfx-card>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import LfxCard from '~/components/uikit/card/card.vue';
import LfxIcon from '~/components/uikit/icon/icon.vue';
import LfxSkeleton from '~/components/uikit/skeleton/skeleton.vue';
import LfxProgressBar from '~/components/uikit/progress-bar/progress-bar.vue';
import { formatNumberShort } from '~/utils/formatter';
import { getCategoryVisual, getCategoryLabel } from '~/config/statistics/categories';
import type { FundingCategory } from '#shared/types/statistics.types';

const props = defineProps<{
  categories: FundingCategory[];
  isLoading: boolean;
  error: Error | null;
}>();

const OTHERS_ID = 'others';

// Categories shown individually in this card; everything else is grouped into "Others".
const MAIN_CATEGORY_KEYS = new Set([
  'general fund',
  'development',
  'marketing',
  'meetups',
  'bugbounty',
  'travel',
  'documentation',
  'security audit',
  'mentorship',
]);

const isMainCategory = (c: FundingCategory): boolean => MAIN_CATEGORY_KEYS.has(c.name.trim().toLowerCase());

const mainCategories = computed((): FundingCategory[] =>
  props.categories.filter(isMainCategory).sort((a, b) => b.raisedCents - a.raisedCents),
);

const othersCategory = computed((): FundingCategory | null => {
  const rest = props.categories.filter((c) => !isMainCategory(c));
  if (!rest.length) return null;
  return {
    id: OTHERS_ID,
    name: 'Others',
    icon: '',
    raisedCents: rest.reduce((sum, c) => sum + c.raisedCents, 0),
    goalCents: 0,
    supporterCount: rest.reduce((sum, c) => sum + c.supporterCount, 0),
  };
});

// Main categories first (largest first), then the combined "Others" bucket always last.
const displayCategories = computed((): FundingCategory[] =>
  othersCategory.value ? [...mainCategories.value, othersCategory.value] : mainCategories.value,
);

const categoryColor = (name: string): string => getCategoryVisual(name).color;

const categoryBg = (name: string) => `background-color: ${categoryColor(name)}`;

const total = () => props.categories.reduce((sum, c) => sum + c.raisedCents, 0);

const categoryBarPercent = (c: FundingCategory): number => {
  const t = total();
  return t > 0 ? (c.raisedCents / t) * 100 : 0;
};

const categoryPercent = (c: FundingCategory): string => {
  const pct = categoryBarPercent(c);
  if (pct === 0) return '0';
  // 1%+ reads as a whole number; below that, show one decimal, rounding up
  // so small-but-real shares aren't flattened to 0%.
  if (Math.abs(pct) >= 1) return String(Math.round(pct));
  const rounded = Math.ceil(Math.abs(pct) * 10) / 10;
  return `${pct < 0 ? '-' : ''}${rounded.toFixed(1)}`;
};

const formatShort = (cents: number) => `$${formatNumberShort(cents / 100)}`;

const categoryTooltip = (c: FundingCategory): string =>
  `${getCategoryLabel(c.name)} · ${categoryPercent(c)}% · ${formatShort(c.raisedCents)} donated`;
</script>

<script lang="ts">
export default {
  name: 'StatisticsFundingByCategory',
};
</script>
