<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="pt-18 pb-16 flex flex-col gap-10">
    <!-- Eyebrow -->
    <div class="flex flex-col gap-5">
      <div class="flex items-center gap-2">
        <lfx-icon
          name="laptop-code"
          type="light"
          :size="20"
          class="text-accent-800"
        />
        <span class="text-lg font-medium leading-7 text-accent-800">For Projects</span>
      </div>

      <h1 class="text-5xl font-light leading-tight text-black">Raise funds for your open source project</h1>

      <p class="text-base font-normal leading-6 text-neutral-900">
        Whether you need to fund a security audit, support infrastructure, sponsor travel, or grow your community — LFX
        CrowdFunding gives you the tools, trust, and transparency to make it happen.
      </p>
    </div>

    <!-- CTAs -->
    <div class="flex items-center gap-6">
      <NuxtLink :to="AppRoute.StartFundraise">
        <lfx-button
          icon="box-dollar"
          label="Start a Fundraise"
          type="primary"
          button-style="pill"
          size="small"
        />
      </NuxtLink>
      <NuxtLink :to="AppRoute.Initiatives">
        <lfx-button
          icon="folder-heart"
          label="Explore Initiatives"
          type="transparent"
          button-style="pill"
          size="small"
        />
      </NuxtLink>
    </div>
  </div>
</template>

<script setup lang="ts">
import LfxIcon from '~/components/uikit/icon/icon.vue';
import LfxButton from '~/components/uikit/button/button.vue';
import { AppRoute } from '~/config/routes';
</script>

<script lang="ts">
export default {
  name: 'ForProjectsHero',
};
</script>
