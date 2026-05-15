<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="border-t border-neutral-200 flex gap-8 items-start md:flex-row flex-col">
    <!-- Left sidebar -->
    <div class="md:basis-1/3 flex items-start md:py-16 pt-10">
      <div class="flex items-center gap-5">
        <img
          :src="bullet"
          alt=""
          class="h-8 w-8"
        />
        <h2 class="text-xl font-semibold leading-8 text-neutral-900 whitespace-nowrap">Get Started</h2>
      </div>
    </div>

    <!-- Right content -->
    <div class="md:basis-2/3 flex flex-col gap-10 md:py-16 pb-16">
      <div class="flex flex-col gap-8">
        <div
          v-for="step in steps"
          :key="step.number"
          class="flex gap-6 items-start"
        >
          <div class="size-6 rounded-full border border-accent-500 flex items-center justify-center shrink-0 mt-0.5">
            <span class="text-xs font-semibold text-accent-500 leading-4">{{ step.number }}</span>
          </div>
          <div class="flex flex-col gap-1 flex-1 min-w-0">
            <p class="text-base font-semibold leading-6 text-neutral-900">{{ step.title }}</p>
            <p class="text-base font-normal leading-6 text-neutral-900">{{ step.description }}</p>
          </div>
        </div>
      </div>

      <NuxtLink
        :to="AppRoute.ForProjects"
        class="md:w-auto w-full"
      >
        <lfx-button
          icon="laptop-code"
          label="Learn more for Projects"
          type="primary"
          button-style="pill"
          class="w-full justify-center"
        />
      </NuxtLink>
    </div>
  </div>
</template>

<script setup lang="ts">
import LfxButton from '~/components/uikit/button/button.vue';
import bullet from '~/assets/images/get-started-bullet.svg';
import { AppRoute } from '~/config/routes';

const steps = [
  {
    number: 1,
    title: 'Check eligibility',
    description: 'Your project should be an open source project with an active community and a clear funding need.',
  },
  {
    number: 2,
    title: 'Review our terms',
    description: 'Understand the benefits and terms of fiscal hosting through the Linux Foundation.',
  },
  {
    number: 3,
    title: 'Start a fundraise',
    description:
      'Create your initiative, set your funding goal, and start accepting donations from individuals and organizations.',
  },
];
</script>

<script lang="ts">
export default {
  name: 'AboutGetStarted',
};
</script>
