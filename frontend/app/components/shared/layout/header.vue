<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <header class="sticky top-0 z-50 border-b border-neutral-100 bg-white">
    <div class="container flex items-center justify-between px-5 py-4 md:px-10">
      <!-- Left: hamburger/app-switcher + logo + desktop nav -->
      <div class="flex items-center gap-2 md:gap-10">
        <div class="flex items-center gap-2 md:gap-4">
          <!-- Mobile: hamburger -->
          <lfx-icon-button
            type="transparent"
            icon="bars"
            icon-type="light"
            :icon-size="18"
            aria-label="Open menu"
            class="visible md:!hidden"
            @click="mobileMenuOpen = true"
          />
          <!-- Desktop: app-switcher -->
          <lfx-tools class="hidden md:block" />

          <NuxtLink
            :to="AppRoute.Home"
            class="flex items-center gap-2"
          >
            <img
              src="~/assets/images/logo.svg"
              alt="LFX"
              class="h-6"
            />
            <span class="text-2xl font-normal text-brand-500">Crowdfunding</span>
          </NuxtLink>
        </div>

        <!-- Desktop nav -->
        <lfx-desktop-nav />
      </div>

      <!-- Right: CTAs (desktop) + divider (desktop) + user -->
      <div class="flex items-center gap-4">
        <div class="hidden items-center gap-4 md:flex">
          <lfx-button
            label="Start Fundraise"
            type="transparent"
            button-style="pill"
            icon="box-dollar"
          />
          <lfx-button
            label="Donate"
            type="primary"
            button-style="pill"
            icon="hand-heart"
          />
        </div>

        <div class="hidden h-6 w-px bg-neutral-200 md:block" />

        <!-- User login -->
        <lfx-user-login />
      </div>
    </div>

    <!-- Mobile menu drawer -->
    <lfx-mobile-menu v-model="mobileMenuOpen" />
  </header>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { AppRoute } from '~/config/routes';
import LfxButton from '~/components/uikit/button/button.vue';
import LfxIconButton from '~/components/uikit/icon-button/icon-button.vue';
import LfxDesktopNav from '~/components/shared/layout/components/desktop-nav.vue';
import LfxMobileMenu from '~/components/shared/layout/components/mobile-menu.vue';
import LfxUserLogin from '~/components/shared/layout/components/user-login.vue';
import LfxTools from '~/components/shared/layout/tools.vue';

const mobileMenuOpen = ref(false);
</script>

<script lang="ts">
export default {
  name: 'CrowdfundingHeader',
};
</script>
