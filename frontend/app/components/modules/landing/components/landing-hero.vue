<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <section class="container pt-18 pb-16 flex flex-col md:gap-16 gap-0">
    <!-- Top row: copy (left) + ring stat (right) -->
    <div class="flex items-center justify-between gap-10 md:flex-row flex-col">
      <!-- Left: badges + headline + subtitle -->
      <div class="flex flex-col gap-10 justify-center md:basis-4/6 w-full">
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
            <span class="text-xs text-neutral-600 leading-4">Trusted by {{ supporterCount }} supporters</span>
          </div>
        </div>

        <!-- Headline + subtitle -->
        <div class="flex flex-col gap-5">
          <h1 class="font-secondary font-light md:text-5xl text-4xl leading-normal text-neutral-900">
            Fund the open source<br />
            <span class="text-accent-500 font-secondary font-light">software that powers</span> the world
          </h1>
          <p class="text-base text-neutral-900 leading-6">
            Non-profit crowdfunding platform sustaining the open source ecosystem.
          </p>
        </div>
        <!-- Trust signals -->
        <trust-badge class="md:hidden flex flex-col !items-start w-full" />
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
            <span class="text-[60px] font-normal text-neutral-900 leading-[72px]">{{ totalRaisedDollars }}</span>
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
      <trust-badge class="md:flex hidden" />

      <!-- CTAs -->
      <div class="flex md:flex-row flex-col md:items-center items-stretch md:w-auto w-full gap-6">
        <start-fundraise-button
          type="tertiary"
          class="md:justify-start justify-center"
        />
        <NuxtLink :to="AppRoute.Initiatives">
          <lfx-button
            label="Explore initiatives"
            type="primary"
            button-style="pill"
            icon="hand-heart"
            class="md:justify-start justify-center"
          />
        </NuxtLink>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import TrustBadge from './trust-badge.vue';
import LfxButton from '~/components/uikit/button/button.vue';
import StartFundraiseButton from '~/components/shared/components/start-fundraise-button.vue';
import LfxAvatar from '~/components/uikit/avatar/avatar.vue';
import LfxAvatarGroup from '~/components/uikit/avatar-group/avatar-group.vue';
import LfxDonutChart from '~/components/uikit/donut-chart/donut-chart.vue';
import { useStatisticsOverview } from '~/composables/statistics/useStatisticsOverview';
import { AppRoute } from '~/config/routes';

const { data: stats } = useStatisticsOverview();

const GOAL = 10_000_000;

const totalRaisedDollars = computed(() => {
  const cents = stats.value?.totalRaisedCents ?? 0;
  return `$${(cents / 100 / 1_000_000).toFixed(1)}M`;
});

const progressPercent = computed(() => {
  const cents = stats.value?.totalRaisedCents ?? 0;
  return Math.min(100, Math.round((cents / 100 / GOAL) * 100));
});

const supporterCount = computed(() => {
  const count = stats.value?.supporterCount ?? 0;
  return count > 0 ? `${count.toLocaleString()}+` : '1,800+';
});

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
