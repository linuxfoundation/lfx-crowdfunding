<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div v-if="isDrawerOpened">
    <teleport to="body">
      <div
        class="c-drawer"
        :class="`c-drawer--${props.position}`"
        @click="clickOutsideClose()"
      >
        <div
          class="c-drawer__content"
          :style="props.position === 'bottom' ? { 'max-height': props.height } : { 'max-width': props.width }"
          v-bind="$attrs"
          @click.stop
        >
          <lfx-icon-button
            v-if="!props.hideCloseButton"
            type="transparent"
            icon="xmark"
            class="absolute top-0 right-0 mr-5 mt-5 z-[999]"
            @click="clickOutsideClose()"
          />
          <slot :close="close" />
        </div>
      </div>
    </teleport>
  </div>
</template>

<script setup lang="ts">
import { computed, onUnmounted, watch } from 'vue';
import LfxIconButton from '~/components/uikit/icon-button/icon-button.vue';

const props = withDefaults(
  defineProps<{
    modelValue: boolean;
    width?: string;
    height?: string;
    closeFunction?: () => boolean;
    position?: 'left' | 'right' | 'bottom';
    hideCloseButton?: boolean;
  }>(),
  {
    width: '37.5rem',
    height: '85vh',
    closeFunction: () => true,
    position: 'right',
    hideCloseButton: false,
  },
);

const emit = defineEmits<{ (e: 'update:modelValue', value: boolean): void }>();

const isDrawerOpened = computed<boolean>({
  get() {
    return props.modelValue;
  },
  set(value: boolean) {
    emit('update:modelValue', value);
  },
});

const close = () => {
  emit('update:modelValue', false);
};

const clickOutsideClose = () => {
  const canClose = props.closeFunction();
  if (canClose) {
    close();
  }
};

const onEscapeKeyUp = (event: KeyboardEvent) => {
  if (event.key === 'Escape') {
    clickOutsideClose();
  }
};

watch(
  () => isDrawerOpened.value,
  (show: boolean) => {
    if (!import.meta.client) return;
    if (!show) {
      window.removeEventListener('keyup', onEscapeKeyUp);
    } else {
      window.addEventListener('keyup', onEscapeKeyUp);
    }
  },
  { immediate: true },
);

onUnmounted(() => {
  if (!import.meta.client) return;
  window.removeEventListener('keyup', onEscapeKeyUp);
});
</script>

<script lang="ts">
export default {
  name: 'LfxDrawer',
};
</script>
