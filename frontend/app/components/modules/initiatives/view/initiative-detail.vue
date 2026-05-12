<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div>
    <template v-if="isLoading">
      <div class="flex items-center justify-center py-24">
        <lfx-spinner :size="32" />
      </div>
    </template>

    <template v-else-if="error">
      <div class="flex items-center justify-center py-24 text-neutral-500 text-sm">Failed to load initiative.</div>
    </template>

    <template v-else-if="data">
      <initiative-detail-header
        :initiative="data"
        :active-tab="activeTab"
        @update:active-tab="activeTab = $event"
      />

      <div class="bg-white py-10">
        <div class="container">
          <div class="flex gap-8 items-start">
            <!-- Left column -->
            <div class="flex-1 min-w-0 flex flex-col gap-8">
              <initiative-detail-funding-card :initiative="data" />
              <initiative-detail-impact
                v-if="data.impactStats?.length"
                :stats="data.impactStats"
              />
              <initiative-detail-project-health
                v-if="data.projectHealthStats?.length"
                :stats="data.projectHealthStats"
                :rating="data.projectHealthRating"
              />
            </div>

            <!-- Right column -->
            <div class="w-[360px] shrink-0 flex flex-col gap-10">
              <initiative-detail-sponsors
                v-if="data.sponsors?.length"
                :sponsors="data.sponsors"
                :initiative-id="data.initiativeId"
              />
              <div class="border-t border-neutral-200 pt-10">
                <recent-donations
                  v-if="data.recentDonations?.length"
                  :donations="data.recentDonations"
                />
              </div>
            </div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import InitiativeDetailHeader from '../components/initiative-detail-header.vue';
import InitiativeDetailFundingCard from '../components/initiative-detail-funding-card.vue';
import InitiativeDetailImpact from '../components/initiative-detail-impact.vue';
import InitiativeDetailProjectHealth from '../components/initiative-detail-project-health.vue';
import InitiativeDetailSponsors from '../components/initiative-detail-sponsors.vue';
import { useInitiative } from '~/composables/useInitiative';
import RecentDonations from '~/components/shared/components/donations/recent-donations.vue';
import LfxSpinner from '~/components/uikit/spinner/spinner.vue';

const props = defineProps<{ initiativeId: string }>();

const { data, isLoading, error } = useInitiative(computed(() => props.initiativeId));
const activeTab = ref('overview');
</script>

<script lang="ts">
export default {
  name: 'InitiativeDetailView',
};
</script>
