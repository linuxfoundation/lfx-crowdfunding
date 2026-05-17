<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <lfx-card
    class="p-6 flex flex-col gap-8"
    :style="{ backgroundImage: cardGradient }"
  >
    <!-- Current balance + progress -->
    <div class="flex flex-col gap-6">
      <div class="flex flex-col gap-1">
        <p class="text-xs font-semibold uppercase tracking-wide text-neutral-500 leading-4">Current Balance</p>
        <p class="text-4xl text-neutral-900 leading-[56px]">
          {{ currentBalanceFormatted }}
        </p>
      </div>

      <div class="flex flex-col gap-2">
        <div class="flex items-center gap-4 text-sm">
          <p class="flex-1 min-w-0 text-neutral-600">
            <span class="font-semibold text-neutral-900">{{ amountRaisedFormatted }} raised</span>
            <span> of {{ goalFormatted }} goal</span>
          </p>
          <p class="text-neutral-500 text-right shrink-0">{{ percentFunded }}% funded</p>
        </div>

        <lfx-progress-bar
          :values="[percentFunded]"
          color="normal"
          size="small"
        />

        <p class="text-sm text-neutral-500">
          {{ supportersLabel }}
        </p>
      </div>
    </div>

    <!-- Funding allocation -->
    <div
      v-if="initiative.fundingGoals?.length"
      class="flex flex-col gap-6"
    >
      <p class="text-base font-semibold text-neutral-900 leading-6">Funding allocation</p>

      <div class="grid md:grid-cols-2 grid-cols-1 gap-8">
        <div
          v-for="goal in initiative.fundingGoals"
          :key="goal.id"
          class="flex gap-4 items-start"
        >
          <!-- Double donut: outer = donated, inner = spent -->
          <lfx-donut-chart
            :size="68"
            :value="donatedPct(goal)"
            color="#009aff"
            :stroke-width="4"
            class="shrink-0"
          >
            <lfx-donut-chart
              :size="54"
              :value="spentPct(goal)"
              color="#002741"
              :stroke-width="4"
            />
          </lfx-donut-chart>

          <div class="flex md:gap-4 gap-3 items-start w-full md:flex-row flex-col">
            <!-- Info -->
            <div class="flex flex-1 min-w-0 flex-col gap-2">
              <p class="text-sm font-semibold text-neutral-900 leading-5">
                {{ goal.name }}
              </p>
              <div class="flex md:flex-col flex-row gap-2">
                <div class="flex items-center gap-1">
                  <lfx-icon
                    name="circle-small"
                    type="solid"
                    :size="12"
                    class="text-accent-500 shrink-0"
                  />
                  <span class="text-xs font-semibold text-neutral-900 leading-4">Donated</span>
                  <span class="text-xs text-neutral-600 leading-4">{{ formatAmount(goal.donatedCents) }}</span>
                </div>
                <div class="flex items-center gap-1">
                  <lfx-icon
                    name="circle-small"
                    type="solid"
                    :size="12"
                    class="text-[#002741] shrink-0"
                  />
                  <span class="text-xs font-semibold text-neutral-900 leading-4">Spent</span>
                  <span class="text-xs text-neutral-600 leading-4">{{ formatAmount(goal.spentCents) }}</span>
                </div>
              </div>
            </div>

            <p class="text-sm text-neutral-500 leading-5 text-right shrink-0 whitespace-nowrap">
              Goal: {{ formatAmount(goal.goalCents) }}
            </p>
          </div>
        </div>
      </div>
    </div>
  </lfx-card>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import {
  initiativeTypeConfigMap,
  defaultInitiativeTypeConfig,
} from '../../../../shared/components/initiative-card/initiative-card.config';
import LfxCard from '~/components/uikit/card/card.vue';
import type { InitiativeDetail, FundingGoal } from '#shared/types/initiative-detail.types';
import LfxProgressBar from '~/components/uikit/progress-bar/progress-bar.vue';
import LfxIcon from '~/components/uikit/icon/icon.vue';
import LfxDonutChart from '~/components/uikit/donut-chart/donut-chart.vue';

const props = defineProps<{ initiative: InitiativeDetail }>();

const typeConfig = computed(
  () => initiativeTypeConfigMap[props.initiative.initiativeType] ?? defaultInitiativeTypeConfig,
);

const cardGradient = computed(() => typeConfig.value.gradient);

const formatAmount = (cents: number): string => {
  const dollars = cents / 100;
  if (dollars >= 1_000_000) return `$${(dollars / 1_000_000).toFixed(1).replace(/\.0$/, '')}M`;
  if (dollars >= 1_000) return `$${Math.round(dollars / 1_000)}K`;
  return `$${dollars.toLocaleString()}`;
};

const currentBalanceFormatted = computed(() => formatAmount(props.initiative.currentBalanceCents ?? 0));

const amountRaisedFormatted = computed(() => formatAmount(props.initiative.fundingStatus?.amountRaisedCents ?? 0));

const goalFormatted = computed(() => formatAmount(props.initiative.fundingStatus?.goalsTotalCents ?? 0));

const percentFunded = computed(() => {
  const goal = props.initiative.fundingStatus?.goalsTotalCents ?? 0;
  const raised = props.initiative.fundingStatus?.amountRaisedCents ?? 0;
  return goal > 0 ? Math.min(100, Math.round((raised / goal) * 100)) : 0;
});

const supportersLabel = computed(() => {
  const count = props.initiative.initiativeStats?.supporters ?? 0;
  return `${count.toLocaleString()} supporters`;
});

const donatedPct = (goal: FundingGoal) =>
  goal.goalCents > 0 ? Math.min(100, (goal.donatedCents / goal.goalCents) * 100) : 0;

const spentPct = (goal: FundingGoal) =>
  goal.goalCents > 0 ? Math.min(100, (goal.spentCents / goal.goalCents) * 100) : 0;
</script>

<script lang="ts">
export default {
  name: 'InitiativeDetailFundingCard',
};
</script>
