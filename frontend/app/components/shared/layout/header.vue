<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <header class="sticky top-0 z-50 border-b border-neutral-100 bg-white">
    <div class="container flex items-center justify-between px-10 py-4">
      <!-- Left: app-switcher + logo + nav -->
      <div class="flex items-center gap-10">
        <div class="flex items-center gap-4">
          <lfx-tools />
          <NuxtLink
            to="/"
            class="flex items-center gap-2"
          >
            <img
              src="~/assets/images/logo.svg"
              alt="LFX"
              class="h-6"
            />
            <span class="text-2xl font-normal text-brand-500">Crowdfunding</span>
          </NuxtLink>
        </div>

        <nav class="hidden items-center gap-3 md:flex">
          <!-- Regular link items -->
          <NuxtLink
            v-for="item in regularMenuItems"
            :key="item.label"
            :to="item.to!"
            class="inline-flex items-center gap-2 rounded-full px-3 py-2 text-sm font-medium text-neutral-500 hover:bg-neutral-50 hover:text-neutral-700"
            active-class="!bg-neutral-100 !text-neutral-900"
          >
            <lfx-icon
              :name="item.icon"
              type="light"
              :size="16"
            />
            {{ item.label }}
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
                size="small"
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
      </div>

      <!-- Right: CTAs + divider + user -->
      <div class="flex items-center gap-4">
        <lfx-button
          label="Start Fundraise"
          type="transparent"
          button-style="pill"
          icon="box-dollar"
          size="small"
        />
        <lfx-button
          label="Donate"
          type="primary"
          button-style="pill"
          icon="hand-heart"
          size="small"
        />
        <div class="h-6 w-px bg-neutral-200" />

        <!-- User button + popover -->
        <lfx-popover
          placement="bottom-end"
          aria-label="User menu"
        >
          <lfx-avatar
            v-if="user"
            type="member"
            :src="user.avatarUrl"
            size="small"
            class="cursor-pointer"
          />
          <lfx-icon
            v-else
            name="circle-user"
            type="light"
            :size="20"
            class="cursor-pointer"
          />

          <template #content>
            <div class="c-dropdown w-60">
              <NuxtLink
                to="/my-donations"
                class="c-dropdown__item"
              >
                <lfx-icon
                  name="circle-dollar-to-slot"
                  type="light"
                  :size="16"
                />
                My donations
              </NuxtLink>
              <NuxtLink
                to="/my-initiatives"
                class="c-dropdown__item"
              >
                <lfx-icon
                  name="folder-heart"
                  type="light"
                  :size="16"
                />
                My initiatives
              </NuxtLink>
              <div class="c-dropdown__separator" />
              <div class="c-dropdown__item">
                <lfx-icon
                  name="arrow-right-from-bracket"
                  type="light"
                  :size="16"
                />
                Sign out
              </div>
            </div>
          </template>
        </lfx-popover>
      </div>
    </div>
  </header>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import LfxButton from '~/components/uikit/button/button.vue';
import LfxIcon from '~/components/uikit/icon/icon.vue';
import LfxTools from '~/components/shared/layout/tools.vue';
import LfxAvatar from '~/components/uikit/avatar/avatar.vue';
import LfxDropdown from '~/components/uikit/dropdown/dropdown.vue';
import LfxDropdownGroupTitle from '~/components/uikit/dropdown/dropdown-group-title.vue';
import LfxDropdownSeparator from '~/components/uikit/dropdown/dropdown-separator.vue';
import LfxPopover from '~/components/uikit/popover/popover.vue';
import { lfxHeaderMenu } from '~/config/menu/header';

// ---------------------------------------------------------------------------
// Mock auth — change MOCK_LOGGED_IN to true to preview the logged-in state
// ---------------------------------------------------------------------------
const MOCK_LOGGED_IN = false;
const mockUser = { name: 'Jane Doe', avatarUrl: 'https://i.pravatar.cc/100' };
const user = MOCK_LOGGED_IN ? mockUser : null;
// ---------------------------------------------------------------------------

const regularMenuItems = computed(() => lfxHeaderMenu.filter((i) => !i.children));
const moreItem = computed(() => lfxHeaderMenu.find((i) => !!i.children));

const moreOpen = ref(false);
</script>

<script lang="ts">
export default {
  name: 'CrowdfundingHeader',
};
</script>
