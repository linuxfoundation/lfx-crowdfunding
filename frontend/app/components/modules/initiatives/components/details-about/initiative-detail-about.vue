<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <lfx-card class="p-6 flex flex-col gap-8">
    <p class="text-base font-semibold text-neutral-900 leading-6">About this initiative</p>

    <!-- eslint-disable-next-line vue/no-v-html -->
    <div
      class="lfx-rich-text"
      v-html="renderedDescription"
    />
  </lfx-card>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import LfxCard from '~/components/uikit/card/card.vue';
import type { InitiativeDetail } from '#shared/types/initiative-detail.types';
import { useSanitize } from '~/composables/useSanitize';

const props = defineProps<{ initiative: InitiativeDetail }>();

const { renderDescription } = useSanitize();

const renderedDescription = computed(() => renderDescription(props.initiative.description ?? ''));
</script>

<script lang="ts">
export default {
  name: 'InitiativeDetailAbout',
};
</script>
