<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="border-t border-neutral-200 py-16 flex flex-col gap-10">
    <div class="flex items-start gap-3">
      <div class="flex-1 flex flex-col gap-3 min-w-0">
        <h2 class="md:text-2xl text-xl font-semibold leading-9 text-neutral-900">How LFX CrowdFunding works</h2>
        <p class="md:text-base text-sm font-normal leading-6 text-neutral-900">
          Get started in minutes. No long applications, no gatekeeping.
        </p>
      </div>
      <div class="md:block hidden">
        <start-fundraise-button
          type="primary"
          class="shrink-0"
        />
      </div>
    </div>

    <div class="flex gap-10 items-start md:flex-row flex-col">
      <div
        v-for="step in STEPS"
        :key="step.number"
        class="flex-1 flex flex-col gap-6 min-w-0"
      >
        <!-- Placeholder image -->
        <div
          class="w-full h-36 rounded-xl"
          :class="step.gradientClass"
        />

        <div class="flex flex-col gap-2">
          <ol
            class="text-base font-semibold leading-6 text-neutral-900 list-decimal"
            :start="step.number"
          >
            <li class="ms-6">{{ step.title }}</li>
          </ol>
          <p class="text-sm font-normal leading-5 text-neutral-600">{{ step.description }}</p>
        </div>
      </div>
    </div>

    <div class="md:hidden block">
      <start-fundraise-button
        type="primary"
        class="shrink-0 w-full justify-center"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import StartFundraiseButton from '~/components/shared/components/start-fundraise-button.vue';

const STEPS = [
  {
    number: 1,
    gradientClass: 'bg-gradient-to-b from-white to-accent-100',
    title: 'Create your initiative',
    description:
      'Set your funding goal, describe your project, and define sponsor tiers with custom benefits. Choose from four fund types — Projects, OSTIF Security Audit, General Funds, or Events — and launch in minutes.',
  },
  {
    number: 2,
    gradientClass: 'bg-gradient-to-b from-white to-accent-50',
    title: 'Share with your community',
    description:
      'Promote your initiative to your contributor base, corporate users, and the broader open source ecosystem. Each initiative gets a dedicated page with progress tracking, sponsor tiers, and a shareable link.',
  },
  {
    number: 3,
    gradientClass: 'bg-gradient-to-b from-white to-positive-50',
    title: 'Receive funds transparently',
    description:
      'Funds are disbursed within 5 days. Every donation and expense is publicly visible on your initiative page, building trust with your supporters and accountability with your community.',
  },
];
</script>

<script lang="ts">
export default {
  name: 'ForProjectsHowItWorks',
};
</script>
