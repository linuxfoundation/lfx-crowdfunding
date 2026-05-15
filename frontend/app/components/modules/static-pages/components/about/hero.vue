<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="pb-16 flex flex-col gap-5">
    <div class="flex items-center gap-2">
      <lfx-icon
        name="circle-info"
        type="light"
        :size="20"
        class="text-accent-800"
      />
      <span class="text-lg font-medium leading-7 text-accent-800">About LFX Crowdfunding</span>
    </div>

    <h1 class="md:text-5xl text-4xl font-secondary font-light leading-tight text-black md:w-3/4">
      Non-profit fiscal infrastructure for the open source projects the world depends on.
    </h1>
  </div>
</template>

<script setup lang="ts">
import LfxIcon from '~/components/uikit/icon/icon.vue';
</script>

<script lang="ts">
export default {
  name: 'AboutHero',
};
</script>
