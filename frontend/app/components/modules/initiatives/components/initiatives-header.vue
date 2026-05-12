<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <section class="container px-10 pt-21 pb-10 flex flex-col gap-8">
    <!-- Eyebrow + headline -->
    <div class="flex flex-col gap-6">
      <div class="flex items-center gap-2 text-primary-600">
        <lfx-icon
          name="hand-heart"
          type="light"
          :size="20"
        />
        <span class="text-sm font-semibold leading-4">Initiatives</span>
      </div>
      <h1 class="font-secondary font-light text-5xl leading-normal text-neutral-900">Fund the future of open source</h1>
    </div>

    <!-- Search -->
    <lfx-input
      :model-value="searchTerm"
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
    <div class="flex items-center justify-between gap-4">
      <lfx-tabs
        :model-value="activeType"
        :tabs="filterTabs"
        tab-style="pill"
        width-type="inline"
        @update:model-value="$emit('update:activeType', $event)"
      />

      <lfx-dropdown-select
        :model-value="sortBy"
        width="180px"
        placement="bottom-end"
        @update:model-value="$emit('update:sortBy', $event)"
      >
        <template #trigger="{ selectedOption }">
          <lfx-button
            :label="selectedOption?.label ?? 'Sort'"
            type="transparent"
            button-style="pill"
            size="small"
            icon="arrow-up-arrow-down"
          />
        </template>
        <lfx-dropdown-item
          value="recent"
          label="Most recent"
        />
        <lfx-dropdown-item
          value="name"
          label="Name (A–Z)"
        />
        <lfx-dropdown-item
          value="funded"
          label="Most funded"
        />
      </lfx-dropdown-select>
    </div>
  </section>
</template>

<script setup lang="ts">
import LfxIcon from '~/components/uikit/icon/icon.vue';
import LfxInput from '~/components/uikit/input/input.vue';
import LfxTabs from '~/components/uikit/tabs/tabs.vue';
import LfxButton from '~/components/uikit/button/button.vue';
import LfxDropdownSelect from '~/components/uikit/dropdown/dropdown-select.vue';
import LfxDropdownItem from '~/components/uikit/dropdown/dropdown-item.vue';

defineProps<{
  searchTerm: string;
  activeType: string;
  sortBy: string;
}>();

defineEmits<{
  (e: 'update:searchTerm', value: string): void;
  (e: 'update:activeType', value: string): void;
  (e: 'update:sortBy', value: string): void;
}>();

const filterTabs = [
  { value: 'all', label: 'All' },
  { value: 'project', label: 'Projects' },
  { value: 'mentorship', label: 'Mentorships' },
  { value: 'security_audit', label: 'Security Audits' },
  { value: 'event', label: 'Events' },
  { value: 'general_fund', label: 'General Funds' },
];
</script>

<script lang="ts">
export default {
  name: 'InitiativesHeader',
};
</script>
