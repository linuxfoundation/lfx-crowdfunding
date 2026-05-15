<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="border-t border-neutral-200 py-16 flex flex-col gap-10">
    <div class="flex flex-col gap-3">
      <h2 class="md:text-2xl text-xl font-semibold leading-9 text-neutral-900">Portfolio giving</h2>
      <p class="md:text-base text-sm font-normal leading-6 text-neutral-900">
        Don't pick one project — fund a portfolio. Spread your investment across related initiatives for maximum impact.
      </p>
    </div>

    <div class="border border-neutral-200 rounded-xl flex items-stretch md:flex-row flex-col">
      <template
        v-for="(portfolio, index) in portfolios"
        :key="portfolio.id"
      >
        <!-- Vertical divider between columns -->
        <div
          v-if="index > 0"
          class="md:w-px md:h-auto w-full h-px bg-neutral-200 shrink-0"
        />

        <!-- Portfolio card -->
        <div class="flex-1 flex flex-col justify-between gap-8 p-6 min-w-0">
          <!-- Top content -->
          <div class="flex flex-col gap-3">
            <!-- Icon -->
            <div
              class="size-10 rounded-full border border-neutral-200 shadow-sm flex items-center justify-center shrink-0"
            >
              <lfx-icon
                :name="portfolio.icon"
                type="light"
                :size="20"
                class="text-black"
              />
            </div>

            <!-- Title + description + items -->
            <div class="flex flex-col gap-5">
              <div class="flex flex-col gap-2">
                <p class="text-base font-semibold leading-6 text-neutral-900">{{ portfolio.name }}</p>
                <p class="text-sm font-normal leading-5 text-neutral-600">{{ portfolio.description }}</p>
              </div>

              <!-- Includes -->
              <div class="flex flex-col gap-2">
                <p class="text-xs leading-4 font-semibold uppercase tracking-wide text-neutral-400">Includes</p>
                <div
                  v-for="initiative in portfolio.initiatives"
                  :key="initiative.name"
                  class="flex items-start gap-2"
                >
                  <lfx-icon
                    name="check"
                    type="light"
                    :size="16"
                    class="text-accent-500 shrink-0 mt-0.5"
                  />
                  <span class="text-sm font-normal leading-5 text-neutral-900">{{ initiative.name }}</span>
                </div>
              </div>

              <!-- Suggested funding -->
              <div class="flex flex-col gap-2">
                <p class="text-xs leading-4 font-semibold uppercase tracking-wide text-neutral-400">
                  Suggested Funding
                </p>
                <p class="text-sm font-normal leading-5 text-neutral-900">{{ portfolio.suggestedFunding }}</p>
              </div>
            </div>
          </div>

          <!-- Get started link pinned to bottom -->
          <NuxtLink
            :to="AppRoute.Contact"
            class="flex items-center gap-1.5 text-sm font-medium leading-5 text-accent-500"
          >
            Get started
            <lfx-icon
              name="angle-right"
              type="light"
              :size="16"
            />
          </NuxtLink>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import LfxIcon from '~/components/uikit/icon/icon.vue';
import { portfolios } from '~/config/pages/for-companies-portfolio.config';
import { AppRoute } from '~/config/routes';
</script>

<script lang="ts">
export default {
  name: 'ForCompaniesPortfolio',
};
</script>
