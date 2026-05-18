<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="border-t border-neutral-200 shrink-0">
    <div class="w-full py-8 flex items-center justify-between">
      <lfx-button
        type="outline"
        label="Cancel"
        @click="emit('cancel')"
      />

      <div class="flex items-center gap-5">
        <lfx-icon-button
          v-if="step > 0"
          type="outline"
          icon="chevron-left"
          @click="emit('previous')"
        />
        <lfx-button
          type="primary"
          :label="continueLabel"
          icon="angle-right"
          icon-position="right"
          :disabled="!isCurrentStepValid"
          :loading="submitting"
          @click="emit('continue')"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import LfxButton from '~/components/uikit/button/button.vue';
import LfxIconButton from '~/components/uikit/icon-button/icon-button.vue';

defineProps<{
  step: number;
  continueLabel: string;
  isCurrentStepValid: boolean;
  submitting: boolean;
}>();

const emit = defineEmits<{
  (e: 'cancel'): void;
  (e: 'previous'): void;
  (e: 'continue'): void;
}>();
</script>

<script lang="ts">
export default {
  name: 'FundraiseFooter',
};
</script>
