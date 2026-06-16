<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <nav class="hidden items-center gap-3 md:flex">
    <NuxtLink
      v-for="item in regularMenuItems"
      :key="item.label"
      v-slot="{ href, navigate, isActive }"
      :to="item.to!"
      custom
    >
      <a
        :href="href"
        class="hidden items-center gap-2 rounded-full px-3 py-2 text-sm lg:inline-flex"
        :class="
          isActive
            ? 'bg-brand-50 font-semibold text-neutral-900'
            : 'font-medium text-neutral-500 hover:bg-neutral-50 hover:text-neutral-700'
        "
        @click="navigate"
      >
        <lfx-icon
          :name="item.icon"
          :type="isActive ? 'solid' : 'light'"
          :class="{ 'text-brand-500': isActive }"
          :size="16"
        />
        {{ item.label }}
      </a>
    </NuxtLink>

    <!-- More dropdown -->
    <lfx-dropdown
      v-if="moreItem"
      v-model:visibility="moreOpen"
      width="240px"
      placement="bottom-start"
    >
      <template #trigger>
        <lfx-button
          type="nav"
          button-style="pill"
          :class="moreOpen ? '!bg-neutral-50 !text-neutral-600' : ''"
        >
          <lfx-icon
            :name="moreItem.icon"
            type="light"
            :size="16"
          />
          {{ moreItem.label }}
        </lfx-button>
      </template>
      <NuxtLink
        v-for="item in regularMenuItems"
        :key="item.label"
        :to="item.to"
        class="c-dropdown__item lg:!hidden flex"
        active-class="c-dropdown__item--active"
      >
        <lfx-icon
          :name="item.icon"
          type="light"
          :size="16"
        />
        {{ item.label }}
      </NuxtLink>
      <lfx-dropdown-group-title>Solutions</lfx-dropdown-group-title>
      <template
        v-for="(child, idx) in moreItem.children"
        :key="child.label"
      >
        <lfx-dropdown-separator v-if="idx === 2" />
        <NuxtLink
          :to="child.to"
          class="c-dropdown__item"
          active-class="c-dropdown__item--active"
        >
          <lfx-icon
            :name="child.icon"
            type="light"
            :size="16"
          />
          {{ child.label }}
        </NuxtLink>
      </template>
    </lfx-dropdown>
  </nav>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import LfxButton from '~/components/uikit/button/button.vue';
import LfxDropdown from '~/components/uikit/dropdown/dropdown.vue';
import LfxDropdownGroupTitle from '~/components/uikit/dropdown/dropdown-group-title.vue';
import LfxDropdownSeparator from '~/components/uikit/dropdown/dropdown-separator.vue';
import LfxIcon from '~/components/uikit/icon/icon.vue';
import { lfxHeaderMenu } from '~/config/menu/header';

const regularMenuItems = computed(() => lfxHeaderMenu.filter((i) => !i.children));
const moreItem = computed(() => lfxHeaderMenu.find((i) => !!i.children));

const moreOpen = ref(false);
</script>

<script lang="ts">
export default {
  name: 'CrowdfundingDesktopNav',
};
</script>
