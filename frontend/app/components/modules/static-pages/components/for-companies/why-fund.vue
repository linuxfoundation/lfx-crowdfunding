<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="border-t border-neutral-200 py-16 flex flex-col gap-10">
    <div class="flex flex-col gap-3">
      <h2 class="text-2xl font-semibold leading-9 text-neutral-900">Why companies fund through LFX</h2>
      <p class="text-base font-normal leading-6 text-neutral-900">
        The Linux Foundation provides the operational infrastructure so your contribution is secure, compliant, and
        impactful.
      </p>
    </div>

    <div class="grid grid-cols-2 gap-10">
      <div
        v-for="feature in FEATURES"
        :key="feature.icon"
        class="flex flex-col gap-3"
      >
        <div class="size-10 rounded-full border border-neutral-200 shadow-sm flex items-center justify-center shrink-0">
          <lfx-icon
            :name="feature.icon"
            type="light"
            :size="20"
            class="text-black"
          />
        </div>
        <div class="flex flex-col gap-2">
          <p class="text-base font-semibold leading-6 text-neutral-900">{{ feature.title }}</p>
          <p class="text-sm font-normal leading-5 text-neutral-600">{{ feature.description }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import LfxIcon from '~/components/uikit/icon/icon.vue';

const FEATURES = [
  {
    icon: 'badge-percent',
    title: 'Tax-advantaged giving',
    description:
      'The Linux Foundation is a 501(c)(6) nonprofit. Contributions may be tax-deductible as business expenses. We provide full documentation for your finance team.',
  },
  {
    icon: 'file-invoice-dollar',
    title: 'Invoicing & compliance',
    description:
      'Receive formal invoices, PO-based billing, and compliance-ready receipts. We integrate with your existing procurement workflows.',
  },
  {
    icon: 'flag',
    title: 'Brand visibility',
    description:
      'Your logo on initiative pages, homepage leaderboard placement, and co-branded announcements. Show your community you invest in the ecosystem.',
  },
  {
    icon: 'eye',
    title: 'Full transparency',
    description:
      'See exactly where every dollar goes. Real-time expense tracking, quarterly impact reports, and financial audits for every initiative you fund.',
  },
];
</script>

<script lang="ts">
export default {
  name: 'ForCompaniesWhyFund',
};
</script>
