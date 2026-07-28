<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <template v-if="isLoading">
    <div class="flex flex-col gap-4">
      <lfx-skeleton
        height="1rem"
        width="50%"
      />
      <div
        v-for="n in 6"
        :key="n"
        class="flex items-center gap-3"
      >
        <lfx-skeleton
          :rounded="true"
          width="2.5rem"
          height="2.5rem"
        />
        <div class="flex-1 flex flex-col gap-1">
          <lfx-skeleton height="0.875rem" />
          <lfx-skeleton
            height="0.75rem"
            width="70%"
          />
        </div>
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
    <span class="text-sm">Failed to load recent donations.</span>
  </div>

  <recent-donations
    v-else-if="limitedDonations.length"
    :donations="limitedDonations"
    :show-initiative-link="true"
    :show-see-all-link="false"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue';
import LfxSkeleton from '~/components/uikit/skeleton/skeleton.vue';
import LfxIcon from '~/components/uikit/icon/icon.vue';
import RecentDonations from '~/components/shared/components/donations/recent-donations.vue';
import { useStatisticsRecentDonations } from '~/composables/statistics/useStatisticsRecentDonations';

const { data, isLoading, error: rawError } = useStatisticsRecentDonations();
const error = computed(() => rawError.value as Error | null);

// Show at most the 10 most recent donations (data arrives most-recent-first).
const RECENT_DONATIONS_LIMIT = 10;
const limitedDonations = computed(() => (data.value?.data ?? []).slice(0, RECENT_DONATIONS_LIMIT));
</script>

<script lang="ts">
export default {
  name: 'StatisticsRecentDonations',
};
</script>
