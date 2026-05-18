<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="border border-neutral-200 rounded-xl p-6">
    <div class="flex flex-col gap-5">
      <div class="flex flex-col gap-2">
        <h2 class="text-base font-semibold text-neutral-900">Sponsorship Tiers</h2>
        <p class="text-xs text-neutral-900 leading-4">
          Offer different sponsorship levels with unique benefits (e.g. "Platinum Sponsorship" or "Lunch Sponsorship").
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
        Add tier
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import LfxIcon from '~/components/uikit/icon/icon.vue';
</script>

<script lang="ts">
export default {
  name: 'FundraiseEventSponsorshipTiersSection',
};
</script>
