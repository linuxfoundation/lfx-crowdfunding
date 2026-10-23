<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <label
    class="flex items-center justify-between px-3 py-2 gap-2"
    @click.stop
  >
    <lfx-icon
      name="search"
      :size="16"
      class="text-neutral-400"
    />
    <input
      ref="input"
      v-model="model"
      type="text"
      class="!outline-none !shadow-none flex-grow text-sm text-neutral-900 leading-5"
      :placeholder="props.placeholder || 'Search...'"
      @input="changedValue"
    />
    <lfx-icon
      v-if="model.length > 0"
      name="circle-xmark"
      :size="16"
      type="solid"
      class="text-neutral-300 cursor-pointer"
      @click="
        model = '';
        changedValue();
      "
    />
  </label>
</template>

<script setup lang="ts">
import { useDebounceFn } from '@vueuse/core';
import LfxIcon from '~/components/uikit/icon/icon.vue';

const props = defineProps<{
  modelValue: string;
  placeholder?: string;
  lazy?: boolean;
  // Focus the input when the dropdown opens (the panel mounts on open).
  autofocus?: boolean;
}>();

const emit = defineEmits<{ (e: 'update:modelValue', value: string): void }>();

const model = ref(props.modelValue);
const input = ref<HTMLInputElement | null>(null);

onMounted(() => {
  // Deferred to a task, so it runs once the click that opened the dropdown has been
  // handled and Popper has positioned the panel.
  if (props.autofocus) setTimeout(() => input.value?.focus({ preventScroll: true }));
});

watch(
  () => props.modelValue,
  (val) => {
    model.value = val;
  },
);

const debouncedEmit = useDebounceFn((value: string) => {
  emit('update:modelValue', value);
}, 300);

const changedValue = () => {
  if (props.lazy) {
    debouncedEmit(model.value);
  } else {
    emit('update:modelValue', model.value);
  }
};
</script>
