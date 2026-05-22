<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <lfx-modal
    :model-value="modelValue"
    width="30rem"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <section class="p-6">
      <div class="flex justify-between">
        <div>
          <p
            v-if="defaults?.area"
            class="text-body-2 text-neutral-500 pb-1"
          >
            {{ defaults.area }}
          </p>
          <h3 class="text-heading-3 font-bold">Share</h3>
        </div>
        <lfx-icon-button
          icon="close"
          size="small"
          @click="emit('update:modelValue', false)"
        />
      </div>

      <share-actions
        :defaults="defaults"
        @copied="emit('update:modelValue', false)"
      />
    </section>
  </lfx-modal>
</template>

<script setup lang="ts">
import ShareActions from './share-actions.vue';
import LfxModal from '~/components/uikit/modal/modal.vue';
import LfxIconButton from '~/components/uikit/icon-button/icon-button.vue';
import type { ShareData } from '~/types/share.types';

defineProps<{
  modelValue: boolean;
  defaults: ShareData;
}>();

const emit = defineEmits<{ (e: 'update:modelValue', value: boolean): void }>();
</script>

<script lang="ts">
export default {
  name: 'ShareModal',
};
</script>
