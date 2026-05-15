<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <lfx-card class="p-6 flex flex-col gap-8">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-3 md:justify-start justify-between w-full">
        <p class="text-base font-semibold text-neutral-900 leading-6">Project Health</p>
        <lfx-tag
          v-if="rating"
          variation="positive"
          size="small"
        >
          {{ rating }}
        </lfx-tag>
      </div>
      <a
        href="#"
        class="flex items-center gap-1.5 text-sm font-medium text-accent-500 hover:text-accent-600 leading-5 md:visible hidden"
      >
        View on LFX Insights
        <lfx-icon
          name="arrow-up-right"
          type="light"
          :size="16"
        />
      </a>
    </div>

    <!-- Stats -->
    <div class="flex gap-4 items-start md:flex-row flex-col">
      <div
        v-for="stat in stats"
        :key="stat.label"
        class="flex flex-1 min-w-0 flex-col gap-1"
      >
        <div class="flex items-center gap-2">
          <div
            class="size-6 rounded-full border border-neutral-200 bg-white shadow-sm flex items-center justify-center shrink-0"
          >
            <lfx-icon
              :name="stat.icon"
              type="light"
              :size="12"
            />
          </div>
          <span class="text-sm text-neutral-900 leading-5 whitespace-nowrap">{{ stat.label }}</span>
        </div>
        <p class="text-4xl text-neutral-900 leading-[56px]">
          {{ stat.value }}
        </p>
      </div>
    </div>
  </lfx-card>
</template>

<script setup lang="ts">
import type { ProjectHealthStat } from '#shared/types/initiative-detail.types';
import LfxCard from '~/components/uikit/card/card.vue';
import LfxIcon from '~/components/uikit/icon/icon.vue';
import LfxTag from '~/components/uikit/tag/tag.vue';

defineProps<{
  stats: ProjectHealthStat[];
  rating?: string;
}>();
</script>

<script lang="ts">
export default {
  name: 'InitiativeDetailProjectHealth',
};
</script>
