<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <section class="container pt-18 pb-16 flex flex-col gap-16">
    <!-- Top row: copy (left) + ring stat (right) -->
    <div class="flex items-center justify-between gap-10 md:flex-row flex-col">
      <!-- Left: badges + headline + subtitle -->
      <div class="flex flex-col gap-10 justify-center basis-4/6">
        <!-- Chips row -->
        <div class="flex items-center gap-4">
          <!-- "Powered by LF" chip -->
          <div
            class="flex items-center gap-2.5 bg-neutral-50 border border-neutral-200 rounded-full shadow-sm px-3 py-2"
          >
            <span class="text-xs text-neutral-600 leading-4">Powered by</span>
            <img
              src="https://www.linuxfoundation.org/hubfs/lf-stacked-color.svg"
              alt="Linux Foundation"
              class="h-5 w-15"
            />
          </div>

          <!-- Supporters chip -->
          <div class="flex items-center gap-2.5 px-3 py-2 rounded-full">
            <lfx-avatar-group type="member">
              <lfx-avatar
                v-for="(avatar, i) in supporterAvatars"
                :key="i"
                type="member"
                :src="avatar"
                size="small"
              />
            </lfx-avatar-group>
            <span class="text-xs text-neutral-600 leading-4">Trusted by 1,800+ supporters</span>
          </div>
        </div>

        <!-- Headline + subtitle -->
        <div class="flex flex-col gap-5">
          <h1 class="font-secondary font-light text-5xl leading-normal text-neutral-900">
            Fund the open source<br />
            <span class="text-accent-500 font-secondary font-light">software that powers</span> the world
          </h1>
          <p class="text-base text-neutral-900 leading-6">
            Non-profit crowdfunding platform sustaining the open source ecosystem.
          </p>
        </div>
      </div>

      <!-- Right: circular funds-raised stat -->
      <div class="py-10">
        <lfx-donut-chart
          :value="progressPercent"
          :color="{ from: '#009AFF', to: '#10B981' }"
          :stroke-width="8"
        >
          <div class="flex flex-col items-center text-center gap-1">
            <span class="text-sm font-semibold text-neutral-900 leading-6">Funds raised</span>
            <span class="text-[60px] font-normal text-neutral-900 leading-[72px]">$5.8M</span>
            <span class="text-xs text-neutral-600 leading-5 max-w-[184px]">
              Help us reach $10M to sustain the open source ecosystem
            </span>
          </div>
        </lfx-donut-chart>
      </div>
    </div>

    <!-- Bottom bar: trust signals (left) + CTAs (right) -->
    <div class="flex items-center justify-between">
      <!-- Trust signals -->
      <trust-badge />

      <!-- CTAs -->
      <div class="flex items-center gap-6">
        <lfx-button
          label="Start Fundraise"
          type="tertiary"
          button-style="pill"
          icon="box-dollar"
          size="small"
        />
        <lfx-button
          label="Explore initiatives"
          type="primary"
          button-style="pill"
          icon="hand-heart"
          size="small"
        />
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import TrustBadge from './trust-badge.vue';
import LfxButton from '~/components/uikit/button/button.vue';
import LfxAvatar from '~/components/uikit/avatar/avatar.vue';
import LfxAvatarGroup from '~/components/uikit/avatar-group/avatar-group.vue';
import LfxDonutChart from '~/components/uikit/donut-chart/donut-chart.vue';

const TOTAL_RAISED = 5_800_000;
const GOAL = 10_000_000;

const progressPercent = Math.round((TOTAL_RAISED / GOAL) * 100);

const supporterAvatars = [
  'https://i.pravatar.cc/40?img=1',
  'https://i.pravatar.cc/40?img=2',
  'https://i.pravatar.cc/40?img=3',
  'https://i.pravatar.cc/40?img=4',
  'https://i.pravatar.cc/40?img=5',
];
</script>

<script lang="ts">
export default {
  name: 'LandingHero',
};
</script>
