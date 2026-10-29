<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div
    ref="trigger"
    class="c-popover__trigger"
    :class="{ 'is-open': isVisible }"
    v-bind="$attrs"
    @click="handleClick"
    @keydown="handleTriggerKeydown"
  >
    <slot />
  </div>
  <ClientOnly>
    <teleport
      v-if="isVisible && !props.disabled"
      to="body"
    >
      <div
        ref="popover"
        class="c-popover__content"
        :class="{ 'is-modal': props.isModal, [props.popoverClass || '']: props.popoverClass }"
        @click="props.isModal ? closePopover() : null"
      >
        <slot
          name="content"
          :close="closePopover"
        />
      </div>
    </teleport>
  </ClientOnly>
</template>

<script lang="ts" setup>
import { ref, watch, onMounted, onBeforeUnmount, nextTick } from 'vue';
import type { Instance, Placement } from '@popperjs/core';
import { createPopper } from '@popperjs/core';

const props = withDefaults(
  defineProps<{
    placement?: Placement;
    triggerEvent?: 'click' | 'hover';
    visibility?: boolean;
    spacing?: number;
    disabled?: boolean;
    matchWidth?: boolean;
    isModal?: boolean;
    popoverClass?: string;
    allowPassThrough?: boolean;
  }>(),
  {
    placement: 'bottom-start',
    triggerEvent: 'click',
    visibility: false,
    spacing: 4,
    disabled: false,
    matchWidth: false,
    isModal: false,
    popoverClass: '',
    allowPassThrough: false,
  },
);

const emit = defineEmits<{ (e: 'update:visibility', value: boolean): void }>();

const trigger = ref<HTMLElement | null>(null);
const popover = ref<HTMLElement | null>(null);
const popperInstance = ref<Instance | null>(null);
const isVisible = ref(props.visibility);
const closeTimeout = ref<number | null>(null);

watch(
  () => props.visibility,
  (val) => {
    isVisible.value = val;
  },
);
watch(isVisible, (val) => emit('update:visibility', val));

const createPopperInstance = () => {
  if (trigger.value && popover.value) {
    popperInstance.value = createPopper(trigger.value, popover.value, {
      strategy: 'fixed',
      placement: props.placement,
      modifiers: [
        {
          name: 'offset',
          options: {
            offset: [0, props.spacing],
          },
        },
        ...(props.matchWidth
          ? [
              {
                name: 'sameWidth',
                enabled: true,
                phase: 'beforeWrite',
                requires: ['computeStyles'],
                fn: ({ state }) => {
                  Object.assign(state.styles.popper, {
                    width: `${state.rects.reference.width}px`,
                  });
                },
              },
            ]
          : []),
      ],
    });
  }
};

const destroyPopperInstance = () => {
  popperInstance.value?.destroy();
  popperInstance.value = null;
};

const openPopover = async () => {
  isVisible.value = true;
};

const closePopover = () => {
  isVisible.value = false;
};

const handleClick = (e: Event) => {
  if (!props.allowPassThrough && props.triggerEvent === 'click') {
    e.stopPropagation();
    e.preventDefault();
  }
  if (props.triggerEvent === 'click') {
    if (isVisible.value) {
      closePopover();
    } else {
      openPopover();
    }
  }
};

const handleTriggerKeydown = (e: KeyboardEvent) => {
  if (props.triggerEvent !== 'click') return;
  // Native buttons/links already turn Enter/Space into a click.
  const isNative = (e.target as HTMLElement).closest('button, a, input, textarea, select');
  if ((e.key === 'Enter' || e.key === ' ') && !isNative) {
    e.preventDefault();
    handleClick(e);
  } else if (e.key === 'ArrowDown' && !isVisible.value) {
    e.preventDefault();
    openPopover();
  }
};

const focusTrigger = () => {
  trigger.value?.querySelector<HTMLElement>('button, a, input, [tabindex]:not([tabindex="-1"])')?.focus();
};

// keyup + capture so the Escape doesn't also reach a surrounding drawer/modal (they close on window keyup).
const handleEscape = (e: KeyboardEvent) => {
  if (e.key !== 'Escape') return;
  e.stopPropagation();
  closePopover();
  focusTrigger();
};

const handleClickOutside = (e: Event) => {
  if (popover.value && !popover.value.contains(e.target as Node) && !trigger.value?.contains(e.target as Node)) {
    closePopover();
  }
};

const cancelClose = () => {
  if (closeTimeout.value !== null) {
    clearTimeout(closeTimeout.value);
    closeTimeout.value = null;
  }
};

const scheduleClose = () => {
  closeTimeout.value = window.setTimeout(() => {
    closePopover();
  }, 150);
};

onMounted(async () => {
  if (import.meta.client) {
    // If visibility starts true the teleported panel doesn't exist yet — defer
    // until after the next tick so createPopper finds the element in the DOM.
    if (isVisible.value) {
      await nextTick();
      createPopperInstance();
      if (props.triggerEvent === 'click') {
        document.addEventListener('click', handleClickOutside, true);
        document.addEventListener('keyup', handleEscape, true);
      }
    }
    if (props.triggerEvent === 'hover') {
      trigger.value?.addEventListener('mouseenter', openPopover);
      trigger.value?.addEventListener('mouseleave', scheduleClose);
      popover.value?.addEventListener('mouseenter', cancelClose);
      popover.value?.addEventListener('mouseleave', scheduleClose);
    }
  }
});

watch(isVisible, async (visible) => {
  if (visible) {
    await nextTick();
    createPopperInstance();

    if (props.triggerEvent === 'click') {
      document.addEventListener('click', handleClickOutside, true);
      document.addEventListener('keyup', handleEscape, true);
    }

    if (props.triggerEvent === 'hover') {
      popover.value?.addEventListener('mouseenter', cancelClose);
      popover.value?.addEventListener('mouseleave', scheduleClose);
    }
  } else {
    destroyPopperInstance();

    if (props.triggerEvent === 'click') {
      document.removeEventListener('click', handleClickOutside, true);
      document.removeEventListener('keyup', handleEscape, true);
    }

    if (props.triggerEvent === 'hover') {
      popover.value?.removeEventListener('mouseenter', cancelClose);
      popover.value?.removeEventListener('mouseleave', scheduleClose);
    }
  }
});

onBeforeUnmount(() => {
  destroyPopperInstance();

  if (props.triggerEvent === 'click') {
    document.removeEventListener('click', handleClickOutside, true);
    document.removeEventListener('keyup', handleEscape, true);
  }

  if (props.triggerEvent === 'hover') {
    trigger.value?.removeEventListener('mouseenter', openPopover);
    trigger.value?.removeEventListener('mouseleave', scheduleClose);
    popover.value?.removeEventListener('mouseenter', cancelClose);
    popover.value?.removeEventListener('mouseleave', scheduleClose);
  }

  if (closeTimeout.value !== null) {
    clearTimeout(closeTimeout.value);
  }
});

defineExpose({
  closePopover,
  openPopover,
  focusTrigger,
});
</script>

<script lang="ts">
export default {
  name: 'LfxPopover',
};
</script>
