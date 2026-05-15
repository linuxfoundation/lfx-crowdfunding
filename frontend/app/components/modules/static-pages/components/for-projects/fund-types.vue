<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="border-t border-neutral-200 py-16 flex flex-col gap-10">
    <div class="flex flex-col gap-3">
      <h2 class="md:text-2xl text-xl font-semibold leading-9 text-neutral-900">Fund what matters to your project</h2>
      <p class="md:text-base text-sm font-normal leading-6 text-neutral-900">
        LFX CrowdFunding supports four fund types, so you can raise funds for exactly what your project needs.
      </p>
    </div>

    <div class="grid md:grid-cols-2 grid-cols-1 gap-10">
      <div
        v-for="item in FUND_TYPES"
        :key="item.icon"
        class="flex flex-col gap-3"
      >
        <div class="size-10 rounded-full border border-neutral-200 shadow-sm flex items-center justify-center shrink-0">
          <lfx-icon
            :name="item.icon"
            type="light"
            :size="20"
            class="text-black"
          />
        </div>
        <div class="flex flex-col gap-2">
          <p class="text-base font-semibold leading-6 text-neutral-900">{{ item.title }}</p>
          <p class="text-sm font-normal leading-5 text-neutral-600">{{ item.description }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import LfxIcon from '~/components/uikit/icon/icon.vue';

const FUND_TYPES = [
  {
    icon: 'laptop-code',
    title: 'Projects',
    description:
      'Dedicated funding for specific open source projects — sustain maintainers, fund development, and keep your project healthy.',
  },
  {
    icon: 'box-magnifying-glass',
    title: 'OSTIF Security Audit',
    description:
      'Fund independent, third-party security audits through the Open Source Technology Improvement Fund (OSTIF) to harden critical projects.',
  },
  {
    icon: 'piggy-bank',
    title: 'General Funds',
    description:
      'Flexible funding for foundations and communities — cover infrastructure, travel, mentorship, and operational costs from a single pool.',
  },
  {
    icon: 'calendar',
    title: 'Events',
    description: 'Fund conferences, hackathons, meetups, diversity scholarships, and speaker travel programs.',
  },
];
</script>

<script lang="ts">
export default {
  name: 'ForProjectsFundTypes',
};
</script>
