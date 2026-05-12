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
      v-if="initiative.donationRecords?.length"
      :donations="initiative.donationRecords"
    />

    <initiative-detail-expenses-table
      v-if="initiative.expenseRecords?.length"
      :expenses="initiative.expenseRecords"
    />
  </div>
</template>

<script setup lang="ts">
import InitiativeDetailFinancialSummary from './initiative-detail-financial-summary.vue';
import InitiativeDetailDonationsTable from './initiative-detail-donations-table.vue';
import InitiativeDetailExpensesTable from './initiative-detail-expenses-table.vue';
import type { InitiativeDetail } from '#shared/types/initiative-detail.types';

defineProps<{ initiative: InitiativeDetail }>();
</script>

<script lang="ts">
export default {
  name: 'InitiativeDetailFinancials',
};
</script>
