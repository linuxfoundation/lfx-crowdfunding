<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="border-t border-neutral-200 py-16 flex flex-col gap-10">
    <div class="flex flex-col gap-3">
      <h2 class="md:text-2xl text-xl font-semibold leading-9 text-neutral-900">Why projects choose LFX CrowdFunding</h2>
      <p class="md:text-base text-sm font-normal leading-6 text-neutral-900">
        Built for open source, by the Linux Foundation. No hidden fees, no platform lock-in, no surprises.
      </p>
    </div>

    <div class="grid md:grid-cols-3 grid-cols-1 gap-10">
      <div
        v-for="feature in FEATURES"
        :key="feature.icon"
        class="flex md:flex-col flex-row gap-3"
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
    title: '0% fees',
    description:
      'Every dollar goes to your project. The Linux Foundation underwrites all platform and payment processing fees until we reach $10M in funding.',
  },
  {
    icon: 'building-columns',
    title: '501(c)(6) fiscal host',
    description:
      'All funds flow through the Linux Foundation, a 501(c)(6) non-profit. Corporate sponsors get tax-advantaged contributions with full audit trails.',
  },
  {
    icon: 'eye',
    title: 'Full financial transparency',
    description: 'Every donation and expense is publicly visible. Supporters see exactly where their money goes.',
  },
  {
    icon: 'landmark',
    title: 'Backed by the Linux Foundation',
    description:
      'Your initiative is hosted by the most trusted name in open source, giving donors confidence to contribute.',
  },
  {
    icon: 'chart-pyramid',
    title: 'Customizable sponsor tiers',
    description:
      'Create Bronze, Silver, Gold, and Platinum tiers with custom benefits to attract both individual and corporate sponsors.',
  },
  {
    icon: 'file-invoice-dollar',
    title: 'Tax-advantaged donations',
    description:
      'Donations flow through the Linux Foundation (501(c)(6)), enabling corporate sponsors to make tax-advantaged contributions.',
  },
];
</script>

<script lang="ts">
export default {
  name: 'ForProjectsWhyChoose',
};
</script>
