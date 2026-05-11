<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <nav class="sticky top-0 z-50 bg-white border-b border-neutral-200">
    <div class="container">
      <div class="flex items-center justify-between h-16">
        <!-- Logo + nav links -->
        <div class="flex items-center gap-8">
          <NuxtLink
            to="/"
            class="flex items-center gap-2.5"
          >
            <img
              src="~/assets/images/logo.svg"
              alt="LFX"
              class="h-5"
            />
            <span class="font-normal text-2xl text-brand-500">CrowdFunding</span>
          </NuxtLink>

          <div class="hidden md:flex items-center gap-6">
            <NuxtLink
              to="/campaigns"
              class="text-sm text-neutral-600 hover:text-neutral-900 transition-colors"
            >
              Explore Initiatives
            </NuxtLink>
            <NuxtLink
              to="/for-projects"
              class="text-sm text-neutral-600 hover:text-neutral-900 transition-colors"
            >
              For Projects
            </NuxtLink>
            <NuxtLink
              to="/for-companies"
              class="text-sm text-neutral-600 hover:text-neutral-900 transition-colors"
            >
              For Companies
            </NuxtLink>
            <NuxtLink
              to="/about"
              class="text-sm text-neutral-600 hover:text-neutral-900 transition-colors"
            >
              About
            </NuxtLink>
          </div>
        </div>

        <!-- CTAs -->
        <div class="flex items-center gap-3">
          <lfx-button
            label="Start a Fundraise"
            type="tertiary"
            button-style="rounded"
            size="small"
          />
          <lfx-button
            label="Donate"
            type="primary"
            button-style="rounded"
            size="small"
          />
        </div>
      </div>
    </div>
  </nav>
</template>

<script setup lang="ts">
import LfxButton from '~/components/uikit/button/button.vue';
</script>

<script lang="ts">
export default {
  name: 'CrowdfundingHeader',
};
</script>
