<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="pt-16 md:pb-30 pb-20">
    <div
      class="bg-gradient-to-b from-white to-accent-100 rounded-xl md:p-16 p-6 flex justify-between gap-10 md:flex-row flex-col md:items-end items-start"
    >
      <div class="flex-1 flex flex-col gap-3 min-w-0">
        <h2 class="text-3xl font-light leading-11 text-neutral-900">Ready to fund your project?</h2>
        <p class="text-base font-normal leading-6 text-neutral-900">
          Join the open source projects already raising funds on LFX CrowdFunding. It takes just a few minutes to get
          started.
        </p>
      </div>
      <start-fundraise-button
        label="Start a Fundraise"
        type="primary"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import StartFundraiseButton from '~/components/shared/components/start-fundraise-button.vue';
</script>

<script lang="ts">
export default {
  name: 'ForProjectsCta',
};
</script>
