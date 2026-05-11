<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <section class="relative bg-brand-900 text-white overflow-hidden">
    <!-- Layered background gradients -->
    <div class="absolute inset-0 bg-gradient-to-br from-brand-900 via-brand-800 to-brand-900" />
    <div
      class="absolute inset-0"
      style="background: radial-gradient(ellipse at top right, rgba(0, 148, 255, 0.2), transparent 50%)"
    />
    <div
      class="absolute inset-0"
      style="background: radial-gradient(ellipse at bottom left, rgba(16, 185, 129, 0.1), transparent 50%)"
    />

    <!-- Subtle grid pattern -->
    <div
      class="absolute inset-0 opacity-[0.04]"
      style="
        background-image:
          linear-gradient(rgba(255, 255, 255, 0.5) 1px, transparent 1px),
          linear-gradient(90deg, rgba(255, 255, 255, 0.5) 1px, transparent 1px);
        background-size: 64px 64px;
      "
    />

    <!-- Decorative orbs -->
    <div class="absolute -top-24 -right-24 w-96 h-96 rounded-full blur-3xl bg-brand-500/10" />
    <div class="absolute -bottom-32 -left-32 w-[28rem] h-[28rem] rounded-full blur-3xl bg-brand-400/10" />
    <div
      class="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[40rem] h-[40rem] rounded-full blur-3xl bg-brand-500/5"
    />

    <div class="container py-20 sm:py-28 lg:py-32 relative">
      <div class="text-center flex flex-col items-center">
        <!-- Trust badge -->
        <div
          class="inline-flex items-center gap-2.5 px-4 py-1.5 bg-white/10 border border-white/15 rounded-full text-sm font-medium text-brand-200 mb-8"
        >
          By
          <img
            src="https://www.linuxfoundation.org/hubfs/lf-stacked-color.svg"
            alt="Linux Foundation"
            class="h-5 brightness-0 invert opacity-80"
          />
          <span class="w-px h-4 bg-white/20" />
          Trusted by 1,800+ supporters
        </div>

        <!-- Headline -->
        <h1 class="text-4xl sm:text-5xl lg:text-6xl font-bold leading-tight max-w-4xl mb-6 tracking-tight">
          Fund the open source
          <span class="bg-gradient-to-r from-brand-300 via-brand-400 to-positive-500 bg-clip-text text-transparent">
            software that powers
          </span>
          the world
        </h1>

        <p class="text-lg sm:text-xl text-brand-300 max-w-2xl mb-8 leading-relaxed">
          Non-profit crowdfunding platform sustaining the open source ecosystem.
        </p>

        <!-- Platform goal tracker -->
        <div class="max-w-md w-full mb-10">
          <div class="flex items-center justify-between text-sm mb-2">
            <span class="text-brand-200 font-medium">${{ raisedMillions }}M raised</span>
            <span class="text-brand-300">${{ (PLATFORM_GOAL / 1_000_000).toFixed(0) }}M goal</span>
          </div>
          <lfx-progress-bar
            :values="[progressPercent]"
            color="normal"
            size="small"
          />
          <p class="text-xs text-brand-400 mt-1.5 text-center">
            Help us reach $10M to sustain the open source ecosystem
          </p>
        </div>

        <!-- CTAs -->
        <div class="flex flex-wrap gap-4 justify-center">
          <NuxtLink to="/campaigns">
            <lfx-button
              label="Explore Initiatives"
              icon="arrow-right"
              icon-position="right"
              type="secondary"
              button-style="rounded"
              size="large"
            />
          </NuxtLink>
          <NuxtLink to="/start">
            <lfx-button
              label="Start a Fundraise"
              type="ghost"
              button-style="rounded"
              size="large"
              class="!text-white !border-white !border hover:!text-neutral-900"
            />
          </NuxtLink>
        </div>

        <!-- Trust badges -->
        <div class="mt-14 flex flex-wrap items-center justify-center gap-8 text-sm">
          <div
            v-for="(badge, i) in trustBadges"
            :key="badge.label"
            class="flex items-center gap-2.5 text-brand-300"
          >
            <div class="w-8 h-8 rounded-lg bg-white/10 flex items-center justify-center shrink-0">
              <lfx-icon
                :name="badge.icon"
                type="light"
                :size="14"
                class="text-positive-500"
              />
            </div>
            {{ badge.label }}
            <span
              v-if="i < trustBadges.length - 1"
              class="w-px h-5 bg-white/15 hidden sm:block"
            />
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import LfxButton from '~/components/uikit/button/button.vue';
import LfxProgressBar from '~/components/uikit/progress-bar/progress-bar.vue';
import LfxIcon from '~/components/uikit/icon/icon.vue';

const PLATFORM_GOAL = 10_000_000;
const TOTAL_RAISED = 5_820_000;

const raisedMillions = computed(() => (TOTAL_RAISED / 1_000_000).toFixed(1));
const progressPercent = computed(() => Math.min(100, Math.round((TOTAL_RAISED / PLATFORM_GOAL) * 100)));

const trustBadges = [
  { icon: 'circle-dollar', label: '0% fees' },
  { icon: 'building-columns', label: '501(c)(6) fiscal host' },
  { icon: 'eye', label: 'Full financial transparency' },
];
</script>

<script lang="ts">
export default {
  name: 'CrowdfundingHero',
};
</script>
