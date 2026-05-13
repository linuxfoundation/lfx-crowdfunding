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
          name="buildings"
          type="light"
          :size="20"
          class="text-accent-800"
        />
        <span class="text-lg font-medium leading-7 text-accent-800">For Companies</span>
      </div>

      <h1 class="text-5xl font-light leading-tight text-black">Invest in the open source your business depends on</h1>

      <p class="text-base font-normal leading-6 text-neutral-900">
        The Linux Foundation provides the operational infrastructure so your contribution is secure, compliant, and
        impactful. Join the companies making open source sustainable.
      </p>
    </div>

    <!-- CTAs -->
    <div class="flex items-center gap-6">
      <NuxtLink to="/contact">
        <lfx-button
          icon="envelope"
          label="Contact our team"
          type="primary"
          button-style="pill"
          size="small"
        />
      </NuxtLink>
      <NuxtLink to="/initiatives">
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
</script>

<script lang="ts">
export default {
  name: 'ForCompaniesHero',
};
</script>
