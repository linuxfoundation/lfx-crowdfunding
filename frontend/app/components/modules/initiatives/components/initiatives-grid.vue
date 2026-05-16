<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <section class="container pb-16">
    <!-- Loading skeletons -->
    <div
      v-if="isLoading"
      class="grid grid-cols-1 gap-8 sm:grid-cols-2 lg:grid-cols-3"
    >
      <initiative-card-loading
        v-for="n in 9"
        :key="n"
      />
    </div>

    <!-- Error -->
    <div
      v-else-if="error"
      class="flex items-center gap-2 text-negative-600"
    >
      <lfx-icon
        name="circle-exclamation"
        type="solid"
        :size="16"
      />
      <span class="text-body-1">Failed to load initiatives. Please try again.</span>
    </div>

    <!-- Empty state -->
    <div
      v-else-if="!initiatives.length"
      class="flex flex-col items-center justify-center gap-4 py-24 text-neutral-500"
    >
      <lfx-icon
        name="folder-open"
        type="light"
        :size="40"
      />
      <p class="text-base">No initiatives found.</p>
    </div>

    <!-- Initiative cards -->
    <div
      v-else
      class="grid grid-cols-1 gap-8 sm:grid-cols-2 lg:grid-cols-3"
    >
      <NuxtLink
        v-for="initiative in initiatives"
        :key="initiative.id"
        :to="`/initiatives/${initiative.slug}`"
        class="block"
      >
        <initiative-card :initiative="initiative" />
      </NuxtLink>
    </div>
  </section>
</template>

<script setup lang="ts">
import LfxIcon from '~/components/uikit/icon/icon.vue';
import InitiativeCard from '~/components/shared/components/initiative-card/initiative-card.vue';
import InitiativeCardLoading from '~/components/shared/components/initiative-card/initiative-card-loading.vue';
import type { Initiative } from '~/types/initiative.types';

defineProps<{
  initiatives: Initiative[];
  isLoading: boolean;
  error: Error | null;
}>();
</script>

<script lang="ts">
export default {
  name: 'InitiativesGrid',
};
</script>
