<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="flex flex-col gap-8">
    <initiative-detail-financial-summary
      v-if="initiative.financialSummary"
      :summary="initiative.financialSummary"
    />

    <initiative-detail-donations-table
      :donations="donationRecords"
      :is-loading="isLoadingDonations"
      :has-more="hasMoreDonations"
      :is-loading-more="isLoadingMoreDonations"
      @load-more="$emit('load-more-donations')"
    />

    <initiative-detail-expenses-table
      v-if="expenseRecords.length || isLoadingExpenses"
      :expenses="expenseRecords"
      :is-loading="isLoadingExpenses"
      :has-more="hasMoreExpenses"
      :is-loading-more="isLoadingMoreExpenses"
      @load-more="$emit('load-more-expenses')"
    />
  </div>
</template>

<script setup lang="ts">
import InitiativeDetailFinancialSummary from './initiative-detail-financial-summary.vue';
import InitiativeDetailDonationsTable from './initiative-detail-donations-table.vue';
import InitiativeDetailExpensesTable from './initiative-detail-expenses-table.vue';
import type { InitiativeDetail, DonationRecord, ExpenseRecord } from '#shared/types/initiative-detail.types';

defineProps<{
  initiative: InitiativeDetail;
  donationRecords: DonationRecord[];
  isLoadingDonations?: boolean;
  hasMoreDonations?: boolean;
  isLoadingMoreDonations?: boolean;
  expenseRecords: ExpenseRecord[];
  isLoadingExpenses?: boolean;
  hasMoreExpenses?: boolean;
  isLoadingMoreExpenses?: boolean;
}>();

defineEmits<{
  'load-more-donations': [];
  'load-more-expenses': [];
}>();
</script>

<script lang="ts">
export default {
  name: 'InitiativeDetailFinancials',
};
</script>
