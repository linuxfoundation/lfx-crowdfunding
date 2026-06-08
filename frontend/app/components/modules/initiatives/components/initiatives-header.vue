<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <section
    class="container pt-15 pb-10 flex flex-col sticky gap-8 top-8 bg-white z-10"
    :class="{ 'border-b border-neutral-200 !pb-5': isScrolled }"
  >
    <!-- Eyebrow + headline -->
    <Transition name="header-eyebrow">
      <div
        v-show="!isScrolled"
        class="flex flex-col gap-6"
      >
        <div class="flex items-center gap-2 text-primary-600">
          <lfx-icon
            name="folder-heart"
            type="light"
            :size="16"
          />
          <span class="text-lg font-medium leading-7 text-accent-800">Initiatives</span>
        </div>
        <h1 class="font-secondary font-light md:text-5xl text-4xl leading-normal text-neutral-900">
          Fund the future of open source
        </h1>
      </div>
    </Transition>

    <!-- Search -->
    <lfx-input
      :model-value="searchTerm"
      class="!rounded-full"
      placeholder="Search initiatives..."
      @update:model-value="$emit('update:searchTerm', String($event))"
    >
      <template #prefix>
        <lfx-icon
          name="magnifying-glass"
          type="light"
          :size="16"
          class="text-neutral-400"
        />
      </template>
    </lfx-input>

    <!-- Filter tabs + sort -->
    <div class="flex items-center md:justify-between justify-start gap-4">
      <div class="hidden md:block">
        <lfx-tabs
          :model-value="activeType"
          :tabs="INITIATIVE_FILTER_TABS"
          tab-style="pill"
          @update:model-value="$emit('update:activeType', $event)"
        />
      </div>
      <div class="md:hidden block">
        <lfx-dropdown-select
          :model-value="activeType"
          width="180px"
          placement="bottom-end"
          @update:model-value="$emit('update:activeType', $event)"
        >
          <template #trigger="{ selectedOption }">
            <lfx-button
              :label="selectedOption?.label ?? 'All Initiatives'"
              type="outline"
              button-style="pill"
              icon="arrow-up-arrow-down"
            />
          </template>
          <lfx-dropdown-item
            v-for="tab in INITIATIVE_FILTER_TABS"
            :key="tab.value"
            :value="tab.value"
            :label="tab.value === 'all' ? 'All Initiatives' : tab.label"
          />
        </lfx-dropdown-select>
      </div>

      <lfx-dropdown-select
        :model-value="sortBy"
        width="180px"
        placement="bottom-end"
        @update:model-value="$emit('update:sortBy', $event)"
      >
        <template #trigger="{ selectedOption }">
          <lfx-button
            :label="selectedOption?.label ?? DEFAULT_SORT_OPTION.label"
            type="outline"
            button-style="pill"
            icon="arrow-down-wide-short"
          />
        </template>
        <lfx-dropdown-item
          v-for="option in INITIATIVE_SORT_OPTIONS"
          :key="option.value"
          :value="option.value"
          :label="option.label"
        />
      </lfx-dropdown-select>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted } from 'vue';
import {
  INITIATIVE_FILTER_TABS,
  INITIATIVE_SORT_OPTIONS,
  DEFAULT_SORT_OPTION,
} from '../config/initiatives-header.config';
import LfxIcon from '~/components/uikit/icon/icon.vue';
import LfxInput from '~/components/uikit/input/input.vue';
import LfxTabs from '~/components/uikit/tabs/tabs.vue';
import LfxButton from '~/components/uikit/button/button.vue';
import LfxDropdownSelect from '~/components/uikit/dropdown/dropdown-select.vue';
import LfxDropdownItem from '~/components/uikit/dropdown/dropdown-item.vue';
import useScroll from '~/utils/scroll';

defineProps<{
  searchTerm: string;
  activeType: string;
  sortBy: string;
}>();

// Scroll anchoring compensates for header height changes by adjusting scrollTop,
// which oscillates isScrolled. Disabling it prevents that feedback loop.
let prevOverflowAnchor = '';
onMounted(() => {
  prevOverflowAnchor = document.documentElement.style.overflowAnchor;
  document.documentElement.style.overflowAnchor = 'none';
});
onUnmounted(() => {
  document.documentElement.style.overflowAnchor = prevOverflowAnchor;
});

const { scrollTop } = useScroll();
const isScrolled = computed(() => scrollTop.value > 10);

defineEmits<{
  (e: 'update:searchTerm', value: string): void;
  (e: 'update:activeType', value: string): void;
  (e: 'update:sortBy', value: string): void;
}>();
</script>

<script lang="ts">
export default {
  name: 'InitiativesHeader',
};
</script>

<style scoped>
.header-eyebrow-enter-active,
.header-eyebrow-leave-active {
  transition:
    opacity 0.2s ease,
    transform 0.2s ease,
    max-height 0.25s ease;
  max-height: 200px;
  overflow: hidden;
}

.header-eyebrow-enter-from,
.header-eyebrow-leave-to {
  opacity: 0;
  transform: translateY(-8px);
  max-height: 0;
}
</style>
