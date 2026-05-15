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

      <div
        class="bg-white pt-10 pb-30 transition-all ease-linear"
        :class="{ 'pt-18': isScrolled }"
      >
        <div class="container">
          <div class="flex lg:gap-8 gap-20 items-start lg:flex-row flex-col">
            <!-- Left column -->
            <div class="flex-1 min-w-0 flex flex-col gap-8 w-full">
              <initiative-detail-overview
                v-if="activeTab === 'overview'"
                :initiative="data"
              />
              <initiative-detail-financials
                v-else-if="activeTab === 'financials'"
                :initiative="data"
              />
              <initiative-detail-about
                v-else-if="activeTab === 'about'"
                :initiative="data"
              />
            </div>

            <!-- Right column -->
            <div class="md:w-[360px] w-full shrink-0 flex flex-col gap-10">
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
import InitiativeDetailOverview from '../components/details-overview/initiative-detail-overview.vue';
import InitiativeDetailSponsors from '../components/details-overview/initiative-detail-sponsors.vue';
import InitiativeDetailFinancials from '../components/details-financials/initiative-detail-financials.vue';
import InitiativeDetailAbout from '../components/details-about/initiative-detail-about.vue';
import { useInitiative } from '~/composables/initiatives/useInitiative';
import RecentDonations from '~/components/shared/components/donations/recent-donations.vue';
import LfxSpinner from '~/components/uikit/spinner/spinner.vue';
import useScroll from '~/utils/scroll';

const props = defineProps<{ initiativeId: string }>();

const { data, isLoading, error } = useInitiative(computed(() => props.initiativeId));
const activeTab = ref('overview');

const { scrollTop } = useScroll();
const isScrolled = computed(() => scrollTop.value > 10);
</script>

<script lang="ts">
export default {
  name: 'InitiativeDetailView',
};
</script>
