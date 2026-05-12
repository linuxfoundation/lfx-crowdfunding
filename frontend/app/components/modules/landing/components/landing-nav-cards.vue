<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <section class="container px-10 pb-16">
    <div class="border-t border-neutral-200 pt-16 pb-8" />
    <div class="grid grid-cols-1 gap-8 sm:grid-cols-2 lg:grid-cols-3">
      <nuxt-link
        v-for="card in navCards"
        :key="card.id"
        :to="card.href"
        class="flex flex-col justify-between h-80 p-6 rounded-2xl border border-neutral-200 no-underline"
        :style="{ backgroundImage: card.gradient }"
      >
        <!-- Top row: label + chevron -->
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <lfx-icon
              :name="card.icon"
              type="light"
              :size="16"
              class="text-neutral-900"
            />
            <span class="text-base font-medium text-neutral-900 leading-6">{{ card.label }}</span>
          </div>
          <lfx-icon
            name="angle-right"
            type="light"
            :size="24"
            class="text-neutral-400"
          />
        </div>

        <!-- Bottom tagline -->
        <p class="font-secondary font-light text-3xl leading-11 text-black">{{ card.tagline }}</p>
      </nuxt-link>
    </div>
  </section>
</template>

<script setup lang="ts">
import LfxIcon from '~/components/uikit/icon/icon.vue';
import { navCards } from '~/config/pages/nav-cards.config';
</script>

<script lang="ts">
export default {
  name: 'LandingNavCards',
};
</script>
