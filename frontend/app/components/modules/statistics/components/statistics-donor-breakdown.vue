<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <lfx-card class="p-6 flex flex-col gap-6">
    <p class="text-base font-semibold text-neutral-900 leading-6">Donor Breakdown</p>

    <template v-if="isLoading">
      <lfx-skeleton
        height="2.5rem"
        width="8rem"
      />
      <lfx-skeleton height="0.75rem" />
      <lfx-skeleton height="1rem" />
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
      <span class="text-sm">Failed to load donor breakdown.</span>
    </div>

    <template v-else-if="breakdown">
      <div class="flex flex-col gap-8">
        <!-- Average donation -->
        <div class="flex flex-col">
          <p class="text-4xl font-normal leading-[56px] text-neutral-900">{{ avgDonation }}</p>
          <p class="text-sm text-neutral-600">Avg. donation</p>
        </div>

        <!-- Stacked bar + legend -->
        <div class="flex flex-col gap-4">
          <!-- Two-segment bar -->
          <lfx-progress-bar
            :values="[orgPercent, indPercent]"
            :colors="['#002741', '#009aff']"
            :hide-empty="true"
          />

          <!-- Legend -->
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-2">
              <lfx-icon
                name="circle-small"
                type="solid"
                :size="12"
                class="text-[#002741]"
              />
              <span class="text-sm font-semibold text-neutral-900">Organizations</span>
              <span class="text-sm text-neutral-600">{{ orgPercent }}%・{{ orgAmountShort }} donated</span>
            </div>
            <div class="flex items-center gap-2">
              <lfx-icon
                name="circle-small"
                type="solid"
                :size="12"
                class="text-[#009aff]"
              />
              <span class="text-sm font-semibold text-neutral-900">Individuals</span>
              <span class="text-sm text-neutral-600">{{ indPercent }}%・{{ indAmountShort }} donated</span>
            </div>
          </div>
        </div>
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
import { formatNumberCurrency, formatNumberShort } from '~/utils/formatter';
import type { DonorBreakdown } from '#shared/types/statistics.types';

const props = defineProps<{
  breakdown: DonorBreakdown | undefined;
  isLoading: boolean;
  error: Error | null;
}>();

const total = computed(() =>
  props.breakdown ? props.breakdown.organizationsCents + props.breakdown.individualsCents : 0,
);

const orgPercent = computed(() =>
  total.value > 0 ? Math.round((props.breakdown!.organizationsCents / total.value) * 100) : 0,
);

const indPercent = computed(() => (total.value > 0 ? 100 - orgPercent.value : 0));

const avgDonation = computed(() =>
  props.breakdown ? formatNumberCurrency(props.breakdown.avgDonationCents / 100, 'USD') : '',
);

const orgAmountShort = computed(() =>
  props.breakdown ? formatNumberShort(props.breakdown.organizationsCents / 100) : '',
);

const indAmountShort = computed(() =>
  props.breakdown ? formatNumberShort(props.breakdown.individualsCents / 100) : '',
);
</script>

<script lang="ts">
export default {
  name: 'StatisticsDonorBreakdown',
};
</script>
