<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <section class="container">
    <!-- Section header -->
    <div class="flex flex-col gap-3 border-t border-neutral-200 pt-16 pb-8">
      <h2 class="text-xl font-semibold text-neutral-900">Impact Stories</h2>
      <p class="text-base text-neutral-900 leading-6">See what your funding makes possible</p>
    </div>

    <!-- Story cards -->
    <div class="flex flex-wrap items-start gap-8 pb-16">
      <div
        v-for="story in impactStories"
        :key="story.id"
        class="grow basis-80 flex flex-col gap-5 p-6 rounded-2xl border border-neutral-200 bg-white"
      >
        <!-- Header: logo + initiative name -->
        <div class="flex items-center gap-3">
          <lfx-avatar
            type="organization"
            size="normal"
            :src="story.logoSrc"
          />
          <span class="text-base font-semibold text-neutral-900 leading-6">{{ story.initiativeName }}</span>
        </div>

        <!-- Tags -->
        <div class="flex flex-wrap gap-3">
          <lfx-chip
            v-for="tag in story.tags"
            :key="tag.label"
            class="text-neutral-900 text-sm"
          >
            <span class="font-semibold">{{ tag.value }}</span> {{ tag.label }}
          </lfx-chip>
        </div>

        <!-- Divider -->
        <hr class="border-neutral-200" />

        <!-- Quote + attribution -->
        <div class="flex flex-col gap-8">
          <p class="font-secondary font-light text-xl leading-8 text-neutral-900">{{ story.quote }}</p>
          <p class="text-xs font-semibold uppercase tracking-wide text-neutral-500 leading-4">
            {{ story.attribution }}
          </p>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import LfxAvatar from '~/components/uikit/avatar/avatar.vue';
import LfxChip from '~/components/uikit/chip/chip.vue';
import { impactStories } from '~/config/pages/impact-stories.config';
</script>

<script lang="ts">
export default {
  name: 'LandingImpactStories',
};
</script>
