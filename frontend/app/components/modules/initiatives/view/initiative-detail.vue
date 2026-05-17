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
                :donation-records="donationRecords"
                :is-loading-donations="transactionsLoading"
                :expense-records="expenseRecords"
                :is-loading-expenses="expensesLoading"
              />
              <initiative-detail-about
                v-else-if="activeTab === 'about'"
                :initiative="data"
              />
            </div>

            <!-- Right column -->
            <div class="lg:w-[360px] w-full shrink-0 flex flex-col gap-10">
              <initiative-detail-sponsors
                v-if="data.sponsors?.length"
                :sponsors="data.sponsors"
                :initiative-id="data.slug"
              />
              <div
                v-if="data.sponsors?.length"
                class="border-t border-neutral-200"
              />
              <RecentDonations
                :donations="recentDonations"
                :is-loading="transactionsLoading"
              />
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
import { useInitiativeTransactions } from '~/composables/initiatives/useInitiativeTransactions';
import RecentDonations from '~/components/shared/components/donations/recent-donations.vue';
import LfxSpinner from '~/components/uikit/spinner/spinner.vue';
import useScroll from '~/utils/scroll';
import type { RecentDonation, DonationRecord, ExpenseRecord } from '#shared/types/initiative-detail.types';

const props = defineProps<{ initiativeId: string }>();

const { data, isLoading, error } = useInitiative(computed(() => props.initiativeId));
const { data: txnData, isLoading: transactionsLoading } = useInitiativeTransactions(
  computed(() => props.initiativeId),
  'donations',
);
const { data: expenseData, isLoading: expensesLoading } = useInitiativeTransactions(
  computed(() => props.initiativeId),
  'expenses',
  10,
);

const recentDonations = computed<RecentDonation[]>(() =>
  (txnData.value?.data ?? []).map((t) => ({
    id: t.id,
    donorName: t.donorName ?? 'Anonymous',
    donorLogoUrl: t.donorLogoUrl,
    donorType: t.donorType === 'organization' ? 'organization' : 'member',
    amountCents: t.amountCents,
    timeAgo: formatTimeAgo(t.date),
  })),
);

const donationRecords = computed<DonationRecord[]>(() =>
  (txnData.value?.data ?? []).map((t) => ({
    id: t.id,
    date: new Date(t.date).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' }),
    supporterName: t.donorName ?? 'Anonymous',
    supporterLogoUrl: t.donorLogoUrl,
    supporterType: t.donorType === 'organization' ? 'organization' : 'member',
    donorCategory: t.donorType === 'organization' ? 'Company' : 'Individual',
    amountCents: t.amountCents,
  })),
);

const expenseRecords = computed<ExpenseRecord[]>(() =>
  (expenseData.value?.data ?? []).map((t) => ({
    id: t.id,
    date: new Date(t.date).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' }),
    category: t.category ?? 'Other',
    description: t.category ?? 'Other',
    amountCents: t.amountCents,
  })),
);

function formatTimeAgo(dateStr: string): string {
  const diff = Date.now() - new Date(dateStr).getTime();
  const days = Math.floor(diff / 86_400_000);
  if (days === 0) return 'Today';
  if (days === 1) return 'Yesterday';
  if (days < 30) return `${days}d ago`;
  const months = Math.floor(days / 30);
  if (months < 12) return `${months}mo ago`;
  return `${Math.floor(months / 12)}y ago`;
}

const activeTab = ref('overview');

const { scrollTop } = useScroll();
const isScrolled = computed(() => scrollTop.value > 10);
</script>

<script lang="ts">
export default {
  name: 'InitiativeDetailView',
};
</script>
