<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="pb-16 flex flex-col gap-10">
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

      <h1 class="md:text-5xl text-4xl font-light leading-tight text-black">Raise funds for your open source project</h1>

      <p class="text-base font-normal leading-6 text-neutral-900">
        Whether you need to fund a security audit, support infrastructure, sponsor travel, or grow your community — LFX
        CrowdFunding gives you the tools, trust, and transparency to make it happen.
      </p>
    </div>

    <!-- CTAs -->
    <div class="flex md:items-center items-start gap-6 md:flex-row flex-col">
      <start-fundraise-button type="primary" />
      <NuxtLink
        :to="AppRoute.Initiatives"
        class="md:w-auto w-full"
      >
        <lfx-button
          icon="folder-heart"
          label="Explore Initiatives"
          type="transparent"
          button-style="pill"
          class="w-full justify-center"
        />
      </NuxtLink>
    </div>
  </div>
</template>

<script setup lang="ts">
import LfxIcon from '~/components/uikit/icon/icon.vue';
import LfxButton from '~/components/uikit/button/button.vue';
import { AppRoute } from '~/config/routes';
import StartFundraiseButton from '~/components/shared/components/start-fundraise-button.vue';
</script>

<script lang="ts">
export default {
  name: 'ForProjectsHero',
};
</script>
