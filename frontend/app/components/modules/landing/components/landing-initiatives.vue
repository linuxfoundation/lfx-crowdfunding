<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <section class="container px-10 pb-16">
    <!-- Section header -->
    <div class="flex items-center justify-between border-t border-neutral-200 pt-16 pb-8">
      <h2 class="text-xl font-semibold text-neutral-900">Trending initiatives</h2>
      <lfx-button
        label="View all"
        type="transparent"
        button-style="pill"
        size="small"
        icon="angle-right"
        icon-position="right"
      />
    </div>

    <!-- Loading skeletons -->
    <div
      v-if="isLoading"
      class="grid grid-cols-1 gap-8 sm:grid-cols-2 lg:grid-cols-3"
    >
      <initiative-card-loading
        v-for="n in 3"
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
      <span class="text-body-1">Failed to load campaigns. Please try again.</span>
    </div>

    <!-- Initiative cards -->
    <div
      v-else-if="initiatives.length"
      class="grid grid-cols-1 gap-8 sm:grid-cols-2 lg:grid-cols-3"
    >
      <initiative-card
        v-for="initiative in initiatives"
        :key="initiative.id"
        :initiative="initiative"
      />
    </div>
  </section>
</template>

<script setup lang="ts">
import LfxButton from '~/components/uikit/button/button.vue';
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
  name: 'LandingInitiatives',
};
</script>
