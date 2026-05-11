<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div>
    <!-- Hero -->
    <section class="bg-brand-600 text-white py-20 px-4">
      <div class="max-w-4xl mx-auto text-center">
        <h1 class="text-4xl font-bold mb-4">Fund Open Source</h1>
        <p class="text-lg text-brand-100 mb-8">
          Support open source projects, mentorships, and events through the Linux Foundation.
        </p>
        <NuxtLink
          to="/campaigns"
          class="inline-block bg-white text-brand-600 font-semibold px-6 py-3 rounded-lg hover:bg-brand-50 transition-colors"
        >
          Explore Campaigns
        </NuxtLink>
      </div>
    </section>

    <!-- Campaign list -->
    <section class="max-w-6xl mx-auto py-16 px-4">
      <h2 class="text-2xl font-bold mb-8">Active Campaigns</h2>

      <div
        v-if="isLoading"
        class="text-gray-500"
      >
        Loading campaigns...
      </div>

      <div
        v-else-if="error"
        class="text-red-600"
      >
        Failed to load campaigns. Please try again.
      </div>

      <div
        v-else-if="data"
        class="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3"
      >
        <article
          v-for="campaign in data.data"
          :key="campaign.id"
          class="border border-gray-200 rounded-lg p-6 hover:shadow-md transition-shadow"
        >
          <span class="text-xs font-medium uppercase tracking-wide text-brand-600 mb-2 block">
            {{ campaign.type.replace('_', ' ') }}
          </span>
          <h3 class="text-lg font-semibold mb-2">{{ campaign.title }}</h3>
          <p class="text-gray-600 text-sm mb-4 line-clamp-2">{{ campaign.description }}</p>

          <!-- Progress bar -->
          <div class="mb-2">
            <div class="h-2 bg-gray-100 rounded-full overflow-hidden">
              <div
                class="h-full bg-brand-500 rounded-full"
                :style="{ width: `${Math.min(100, (campaign.raisedAmount / campaign.goalAmount) * 100)}%` }"
              />
            </div>
          </div>
          <p class="text-sm text-gray-500">
            <span class="font-semibold text-gray-900">
              {{ campaign.currency }}{{ campaign.raisedAmount.toLocaleString() }}
            </span>
            raised of {{ campaign.currency }}{{ campaign.goalAmount.toLocaleString() }}
          </p>
        </article>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { useCampaigns } from '~/composables/useCampaigns';

useHead({ title: 'Home' });

const { data, isLoading, error } = useCampaigns();
</script>
