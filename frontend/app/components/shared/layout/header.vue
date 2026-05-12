<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <header class="sticky top-0 z-50 border-b border-neutral-200 bg-white">
    <div class="container flex items-center justify-between px-10 py-4">
      <!-- Left: app-switcher + logo + nav -->
      <div class="flex items-center gap-10">
        <div class="flex items-center gap-4">
          <lfx-icon-button
            type="transparent"
            icon="grid-round"
            icon-type="solid"
            :icon-size="18"
            aria-label="App switcher"
          />
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
              <lfx-dropdown-item :to="child.to">
                <lfx-icon
                  :name="child.icon"
                  type="light"
                  :size="16"
                />
                {{ child.label }}
              </lfx-dropdown-item>
            </template>
          </lfx-dropdown>
        </nav>
      </div>

      <!-- Right: CTAs + divider + user -->
      <div class="flex items-center gap-4">
        <lfx-button
          label="Start Fundraise"
          type="tertiary"
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

        <!-- User button + dropdown -->
        <lfx-dropdown
          v-model:visibility="userOpen"
          width="240px"
          placement="bottom-end"
        >
          <template #trigger>
            <lfx-icon-button
              type="transparent"
              size="medium"
              aria-label="User menu"
              :class="user ? 'relative !bg-neutral-50' : ''"
            >
              <img
                v-if="user"
                :src="user.avatarUrl"
                :alt="user.name"
                class="absolute inset-1.5 size-6 rounded-full object-cover"
              />
              <lfx-icon
                v-else
                name="circle-user"
                type="light"
                :size="16"
              />
            </lfx-icon-button>
          </template>

          <lfx-dropdown-item to="/my-donations">
            <lfx-icon
              name="circle-dollar-to-slot"
              type="light"
              :size="16"
            />
            My donations
          </lfx-dropdown-item>
          <lfx-dropdown-item to="/my-initiatives">
            <lfx-icon
              name="folder-heart"
              type="light"
              :size="16"
            />
            My initiatives
          </lfx-dropdown-item>
          <lfx-dropdown-separator />
          <lfx-dropdown-item>
            <lfx-icon
              name="arrow-right-from-bracket"
              type="light"
              :size="16"
            />
            Sign out
          </lfx-dropdown-item>
        </lfx-dropdown>
      </div>
    </div>
  </header>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import LfxButton from '~/components/uikit/button/button.vue';
import LfxIconButton from '~/components/uikit/icon-button/icon-button.vue';
import LfxIcon from '~/components/uikit/icon/icon.vue';
import LfxDropdown from '~/components/uikit/dropdown/dropdown.vue';
import LfxDropdownItem from '~/components/uikit/dropdown/dropdown-item.vue';
import LfxDropdownGroupTitle from '~/components/uikit/dropdown/dropdown-group-title.vue';
import LfxDropdownSeparator from '~/components/uikit/dropdown/dropdown-separator.vue';
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
const userOpen = ref(false);
</script>

<script lang="ts">
export default {
  name: 'CrowdfundingHeader',
};
</script>
