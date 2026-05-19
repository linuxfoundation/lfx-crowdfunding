<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="border-b border-neutral-200 shrink-0">
    <div class="w-full py-6">
      <div class="flex flex-col gap-1">
        <div class="flex items-center gap-2 text-accent-600">
          <lfx-icon
            name="box-dollar"
            type="light"
            :size="15"
          />
          <span class="text-xs font-medium leading-4">Fundraise</span>
        </div>
        <h1 class="font-secondary font-light text-2xl text-black">Start a new initiative</h1>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import LfxIcon from '~/components/uikit/icon/icon.vue';
</script>

<script lang="ts">
export default {
  name: 'FundraiseHeader',
};
</script>
