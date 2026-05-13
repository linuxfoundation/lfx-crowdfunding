<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="pt-16 pb-30">
    <div class="bg-gradient-to-b from-white to-accent-100 rounded-xl p-16 flex items-end justify-between gap-10">
      <div class="flex-1 flex flex-col gap-3 min-w-0">
        <h2 class="text-3xl font-light leading-11 text-neutral-900">Ready to invest in open source?</h2>
        <p class="text-base font-normal leading-6 text-neutral-900">
          Our team will help you design a giving strategy that aligns with your business priorities and maximizes your
          impact.
        </p>
      </div>
      <div class="flex items-center gap-4">
        <NuxtLink
          to="/initiatives"
          class="shrink-0"
        >
          <lfx-button
            icon="folder-heart"
            label="Browse initiatives"
            type="outline"
            button-style="pill"
            size="small"
          />
        </NuxtLink>
        <NuxtLink
          to="/contact"
          class="shrink-0"
        >
          <lfx-button
            icon="envelope"
            label="Contact our team"
            type="primary"
            button-style="pill"
            size="small"
          />
        </NuxtLink>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import LfxButton from '~/components/uikit/button/button.vue';
</script>

<script lang="ts">
export default {
  name: 'ForCompaniesCta',
};
</script>
