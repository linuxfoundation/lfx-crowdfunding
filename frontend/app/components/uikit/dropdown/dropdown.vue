<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <lfx-popover
    ref="popover"
    v-model:visibility="isVisible"
    :placement="props.placement"
    v-bind="$attrs"
    :is-modal="pageWidth < 640"
    :match-width="props.matchWidth"
    :popover-class="props.popoverClass"
    :allow-pass-through="false"
  >
    <slot name="trigger" />

    <template #content>
      <div
        :id="props.panelId"
        ref="panel"
        class="c-dropdown overflow-y-auto"
        :role="props.panelRole"
        :style="{ width: props.width }"
        :class="props.dropdownClass"
        @click="popover.closePopover()"
        @keydown="handlePanelKeydown"
      >
        <slot />
      </div>
    </template>
  </lfx-popover>
</template>

<script setup lang="ts">
import type { Placement } from '@popperjs/core';
import { computed, provide, watch } from 'vue';
import LfxPopover from '~/components/uikit/popover/popover.vue';
import useResponsive from '~/utils/responsive';

const props = withDefaults(
  defineProps<{
    placement?: Placement;
    width?: string;
    visibility?: boolean;
    matchWidth?: boolean;
    dropdownClass?: string;
    popoverClass?: string;
    panelRole?: string;
    panelId?: string;
  }>(),
  {
    placement: 'bottom-start',
    width: 'auto',
    visibility: false,
    matchWidth: false,
    dropdownClass: '',
    popoverClass: '',
    panelRole: undefined,
    panelId: undefined,
  },
);

const emit = defineEmits<{ (e: 'update:visibility', value: boolean): void }>();

const submenuOpen = ref('');

provide('submenuOpen', submenuOpen);

const isVisible = computed({
  get: () => props.visibility,
  set: (value: boolean) => {
    emit('update:visibility', value);
    submenuOpen.value = '';
  },
});

const popover = ref<LfxPopover | null>(null);

const { pageWidth } = useResponsive();

// Roving focus: the search input, options and menu links are the keyboard stops.
const panel = ref<HTMLElement | null>(null);
const navItems = () =>
  Array.from(panel.value?.querySelectorAll<HTMLElement>('input, [role="option"], a.c-dropdown__item') ?? []);

watch(
  panel,
  (el) => {
    if (!el) return;
    const start = el.querySelector<HTMLElement>('input') ?? el.querySelector<HTMLElement>('[aria-selected="true"]');
    (start ?? navItems()[0])?.focus({ preventScroll: true });
  },
  { flush: 'post' },
);

const handlePanelKeydown = (e: KeyboardEvent) => {
  const target = e.target as HTMLElement;
  if (e.key === 'Tab') {
    // The panel is teleported to <body>; send focus back so Tab continues from the trigger.
    popover.value?.closePopover();
    popover.value?.focusTrigger();
    return;
  }
  if (e.key === ' ' && target.getAttribute('role') === 'option') {
    // Only Enter selects: Space's keyup would land on a refocused trigger button and reopen it.
    e.preventDefault();
    return;
  }
  if (e.key === 'Enter' && target.getAttribute('role') === 'option') {
    e.preventDefault();
    target.click(); // selects, then bubbles to the panel's close handler
    popover.value?.focusTrigger();
    return;
  }

  const items = navItems();
  const i = items.indexOf(target);
  const inInput = target.tagName === 'INPUT';
  let next: number | undefined;
  if (e.key === 'ArrowDown') next = Math.min(i + 1, items.length - 1);
  else if (e.key === 'ArrowUp') next = Math.max(i - 1, 0);
  else if (e.key === 'Home' && !inInput) next = 0;
  else if (e.key === 'End' && !inInput) next = items.length - 1;
  if (next === undefined) return;
  e.preventDefault();
  items[next]?.focus();
};
</script>

<script lang="ts">
export default {
  name: 'LfxDropdown',
};
</script>
