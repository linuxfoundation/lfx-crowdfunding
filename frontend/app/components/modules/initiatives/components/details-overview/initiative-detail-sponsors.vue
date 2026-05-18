<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="flex flex-col gap-6">
    <p class="text-base font-semibold text-neutral-900 leading-6">Sponsors</p>

    <div class="flex flex-wrap gap-4">
      <div
        v-for="sponsor in sponsors"
        :key="sponsor.name"
        class="flex flex-col items-center gap-1.5"
      >
        <lfx-avatar
          type="organization"
          size="large"
          :src="sponsor.avatarUrl"
        />
        <span class="text-xs text-neutral-600 leading-[14px] text-center max-w-[48px] truncate">
          {{ sponsor.name }}
        </span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { SponsorEntry } from '#shared/types/initiative-detail.types';
import LfxAvatar from '~/components/uikit/avatar/avatar.vue';

defineProps<{
  sponsors: SponsorEntry[];
  initiativeId: string;
}>();
</script>

<script lang="ts">
export default {
  name: 'InitiativeDetailSponsors',
};
</script>
