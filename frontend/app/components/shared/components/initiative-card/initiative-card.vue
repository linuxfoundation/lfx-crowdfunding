<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div
    class="flex flex-col justify-between border border-neutral-200 rounded-2xl p-6 h-[400px]"
    :style="{ backgroundImage: typeConfig.gradient }"
  >
    <!-- Top section -->
    <div class="flex flex-col gap-6 w-full">
      <lfx-avatar
        :src="initiative.logoUrl"
        type="organization"
        size="xlarge"
      />

      <div class="flex flex-col gap-4 w-full">
        <!-- Type badge -->
        <div
          class="flex items-center gap-2"
          :class="typeConfig.colorClass"
        >
          <lfx-icon
            :name="typeConfig.icon"
            type="light"
            :size="12"
          />
          <span class="text-xs font-semibold leading-4">{{ typeConfig.label }}</span>
        </div>

        <!-- Title + description -->
        <div class="flex flex-col gap-1 w-full">
          <h3 class="text-lg font-semibold text-neutral-900 leading-7 truncate">
            {{ initiative.name }}
          </h3>
          <p class="text-sm text-neutral-600 leading-5 line-clamp-2">
            {{ initiative.description }}
          </p>
        </div>

        <!-- Tags -->
        <div
          v-if="tags.length"
          class="flex flex-wrap gap-2"
        >
          <lfx-chip
            v-for="tag in tags"
            :key="tag"
            type="bordered"
            size="xsmall"
          >
            {{ tag }}
          </lfx-chip>
        </div>
      </div>
    </div>

    <!-- Bottom funding section -->
    <div class="flex flex-col gap-2 w-full">
      <div class="flex items-center justify-between text-sm">
        <span>
          <span class="font-semibold text-neutral-900">{{ amountRaisedFormatted }}</span>
          <span class="text-neutral-500"> / {{ totalGoalFormatted }}</span>
        </span>
        <span class="text-neutral-500">{{ percentFundedLabel }}</span>
      </div>

      <lfx-progress-bar
        :values="[progressPercent]"
        color="normal"
        size="small"
      />

      <p class="text-sm text-neutral-500">{{ supportersLabel }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { initiativeTypeConfigMap, defaultInitiativeTypeConfig } from './initiative-card.config';
import type { Initiative } from '~/types/initiative.types';
import LfxAvatar from '~/components/uikit/avatar/avatar.vue';
import LfxIcon from '~/components/uikit/icon/icon.vue';
import LfxChip from '~/components/uikit/chip/chip.vue';
import LfxProgressBar from '~/components/uikit/progress-bar/progress-bar.vue';

const props = defineProps<{ initiative: Initiative }>();

const typeConfig = computed(
  () => initiativeTypeConfigMap[props.initiative.initiativeType] ?? defaultInitiativeTypeConfig,
);

const tags = computed(() => {
  const industry = props.initiative.industry;
  if (!industry) return [];
  return industry
    .split(',')
    .map((t) => t.trim())
    .filter(Boolean);
});

const progressPercent = computed(() => {
  const goal = props.initiative.fundingStatus?.totalAnnualGoalInCents ?? 0;
  const raised = props.initiative.fundingStatus?.amountRaisedCents ?? 0;
  return goal > 0 ? Math.min(100, Math.round((raised / goal) * 100)) : 0;
});

const formatAmountAbbrev = (cents: number): string => {
  const dollars = cents / 100;
  if (dollars >= 1_000_000) return `$${(dollars / 1_000_000).toFixed(1).replace(/\.0$/, '')}M`;
  if (dollars >= 1_000) return `$${(dollars / 1_000).toFixed(0)}K`;
  return `$${dollars.toLocaleString()}`;
};

const amountRaisedFormatted = computed(() =>
  formatAmountAbbrev(props.initiative.fundingStatus?.amountRaisedCents ?? 0),
);

const totalGoalFormatted = computed(() =>
  formatAmountAbbrev(props.initiative.fundingStatus?.totalAnnualGoalInCents ?? 0),
);

const percentFundedLabel = computed(() => `${progressPercent.value}% funded`);

const supportersLabel = computed(() => {
  const count = props.initiative.initiativeStats?.backers ?? 0;
  return `${count.toLocaleString()} supporters`;
});
</script>

<script lang="ts">
export default {
  name: 'InitiativeCard',
};
</script>
