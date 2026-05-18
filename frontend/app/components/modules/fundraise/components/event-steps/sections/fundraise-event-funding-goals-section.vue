<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="border border-neutral-200 rounded-xl p-6">
    <div class="flex flex-col gap-5">
      <div class="flex flex-col gap-2">
        <h2 class="text-base font-semibold text-neutral-900">Event Funding Goals</h2>
        <p class="text-xs text-neutral-900 leading-4">
          Add specific funding goals for your event (e.g. "Guest Speaker Fund" or "After-Party Reception").
        </p>
      </div>

      <button
        type="button"
        class="flex items-center gap-1.5 text-sm font-medium text-accent-500 hover:text-accent-600 transition-colors w-fit"
      >
        <lfx-icon
          name="plus"
          type="light"
          :size="16"
        />
        Add funding goal
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import LfxIcon from '~/components/uikit/icon/icon.vue';
</script>

<script lang="ts">
export default {
  name: 'FundraiseEventFundingGoalsSection',
};
</script>
