<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="flex-1 flex items-center justify-center py-12">
    <div class="flex flex-col items-center gap-10 max-w-[600px] w-full">
      <div class="flex flex-col items-center gap-6 text-center">
        <!-- Icon circle -->
        <div class="size-20 rounded-full bg-accent-500 flex items-center justify-center shrink-0">
          <lfx-icon
            name="memo-circle-check"
            type="solid"
            :size="32"
            class="text-white"
          />
        </div>

        <!-- Type chip -->
        <div class="border border-neutral-200 rounded-full flex items-center gap-1 px-2.5 py-1 shrink-0">
          <lfx-icon
            :name="config.icon"
            type="light"
            :size="12"
            class="text-neutral-900"
          />
          <span class="text-sm text-neutral-900">{{ config.label }}</span>
        </div>

        <!-- Heading -->
        <h2 class="font-secondary font-light text-2xl text-black leading-9">Initiative submitted with success!</h2>

        <!-- Review info label -->
        <div class="bg-accent-100 rounded-full flex items-center gap-1 px-2 py-1 shrink-0">
          <lfx-icon
            name="info-circle"
            type="solid"
            :size="12"
            class="text-accent-500"
          />
          <span class="text-sm font-semibold text-accent-500">{{ config.reviewMessage }}</span>
        </div>

        <!-- Body text -->
        <p class="text-base text-neutral-600 leading-6">
          Once approved, you can start accepting donations.<br />
          All funds flow through the Linux Foundation 501(c)(6).
        </p>
      </div>

      <!-- CTA -->
      <lfx-button
        type="transparent"
        label="Manage your Initiatives"
        icon="arrow-up-right"
        icon-type="light"
        icon-position="right"
        @click="emit('done')"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { INITIATIVE_TYPE_CONFIG } from '../../config/initiative-types.config';
import type { InitiativeType } from '~/types/fundraise.types';
import LfxIcon from '~/components/uikit/icon/icon.vue';
import LfxButton from '~/components/uikit/button/button.vue';

const DEFAULT_CONFIG = INITIATIVE_TYPE_CONFIG.project;

const props = defineProps<{
  initiativeType: InitiativeType | null;
}>();

const emit = defineEmits<{
  (e: 'done'): void;
}>();

const config = computed(() => (props.initiativeType ? INITIATIVE_TYPE_CONFIG[props.initiativeType] : DEFAULT_CONFIG));
</script>

<script lang="ts">
export default {
  name: 'FundraiseSuccess',
};
</script>
