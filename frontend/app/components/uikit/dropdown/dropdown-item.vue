<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <component
    :is="rootEl"
    class="c-dropdown__item"
    :class="{ 'is-selected': isSelected }"
    v-bind="rootProps"
    @click="handleClick"
  >
    <slot>
      {{ props.label }}
    </slot>
    <div
      v-if="isSelected || props.checkmarkBefore"
      class="flex justify-end"
      :class="props.checkmarkBefore ? 'order-first min-w-4 w-4' : 'flex-grow'"
    >
      <lfx-icon
        v-if="isSelected"
        name="check"
        :size="16"
        class="!text-brand-500"
      />
    </div>
  </component>
</template>

<script setup lang="ts">
import { computed, resolveComponent } from 'vue';
import type { Ref, WritableComputedRef } from 'vue';

const props = defineProps<{
  value?: string;
  label?: string;
  checkmarkBefore?: boolean;
  to?: string;
  href?: string;
}>();

const rootEl = computed(() => {
  if (props.to) return resolveComponent('NuxtLink');
  if (props.href) return 'a';
  return 'div';
});

const rootProps = computed(() => {
  if (props.to) return { to: props.to };
  if (props.href) return { href: props.href, target: '_blank', rel: 'noopener noreferrer' };
  return {};
});

const attrs = useAttrs();
const selectedValue = inject<WritableComputedRef<string>>('selectedValue', ref('') as WritableComputedRef<string>);
const selectedOptionProps = inject<Ref<Record<string, unknown>>>(
  'selectedOptionProps',
  ref<Record<string, unknown>>({}),
);

const isSelected = computed(() => selectedValue && props.value && selectedValue.value === props.value);

const handleClick = () => {
  if (!props.value) return;
  if (selectedOptionProps) {
    selectedOptionProps.value = { ...props, ...attrs };
  }
  if (selectedValue) {
    selectedValue.value = props.value;
  }
};
</script>

<script lang="ts">
export default {
  name: 'LfxDropdownItem',
};
</script>
