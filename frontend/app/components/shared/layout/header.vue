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
          <button
            type="button"
            class="inline-flex items-center rounded-full px-2.5 py-2 text-neutral-400 hover:bg-neutral-50"
            aria-label="App switcher"
          >
            <lfx-icon
              name="grid-round"
              type="solid"
              :size="18"
            />
          </button>
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
          <div
            v-if="moreItem"
            ref="moreRef"
            class="relative"
          >
            <button
              type="button"
              :class="[
                'inline-flex items-center gap-2 rounded-full px-3 py-2 text-sm font-medium',
                moreOpen
                  ? 'bg-neutral-50 text-neutral-600'
                  : 'text-neutral-500 hover:bg-neutral-50 hover:text-neutral-700',
              ]"
              @click="moreOpen = !moreOpen"
            >
              <lfx-icon
                :name="moreItem.icon"
                type="light"
                :size="16"
              />
              {{ moreItem.label }}
            </button>

            <div
              v-show="moreOpen"
              class="absolute left-0 top-full mt-1 w-60 rounded-lg border border-neutral-200 bg-white p-1 shadow-lg"
            >
              <div class="px-3 pb-1 pt-2">
                <p class="text-[10px] font-semibold uppercase tracking-wide text-neutral-400">Solutions</p>
              </div>
              <template
                v-for="(child, idx) in moreItem.children"
                :key="child.label"
              >
                <div
                  v-if="idx === 2"
                  class="my-1 h-px bg-neutral-200"
                />
                <NuxtLink
                  :to="child.to"
                  class="flex items-center gap-2 rounded-md px-3 py-2 text-sm text-neutral-900 hover:bg-neutral-50"
                  @click="moreOpen = false"
                >
                  <lfx-icon
                    :name="child.icon"
                    type="light"
                    :size="16"
                  />
                  {{ child.label }}
                </NuxtLink>
              </template>
            </div>
          </div>
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
        <div
          ref="userRef"
          class="relative"
        >
          <button
            type="button"
            class="relative inline-flex size-9 items-center justify-center rounded-full hover:bg-neutral-50"
            :class="user ? 'bg-neutral-50' : 'text-neutral-900'"
            aria-label="User menu"
            @click="userOpen = !userOpen"
          >
            <template v-if="user">
              <img
                :src="user.avatarUrl"
                :alt="user.name"
                class="absolute inset-1.5 size-6 rounded-full object-cover"
              />
            </template>
            <template v-else>
              <lfx-icon
                name="circle-user"
                type="light"
                :size="16"
              />
            </template>
          </button>

          <div
            v-show="userOpen"
            class="absolute right-0 top-full mt-1 w-60 rounded-lg border border-neutral-200 bg-white p-1 shadow-lg"
          >
            <NuxtLink
              to="/my-donations"
              class="flex items-center gap-2 rounded-md px-3 py-2 text-sm text-neutral-900 hover:bg-neutral-50"
              @click="userOpen = false"
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
              class="flex items-center gap-2 rounded-md px-3 py-2 text-sm text-neutral-900 hover:bg-neutral-50"
              @click="userOpen = false"
            >
              <lfx-icon
                name="folder-heart"
                type="light"
                :size="16"
              />
              My initiatives
            </NuxtLink>
            <div class="my-1 h-px bg-neutral-200" />
            <button
              type="button"
              class="flex w-full items-center gap-2 rounded-md px-3 py-2 text-sm text-neutral-900 hover:bg-neutral-50"
              @click="userOpen = false"
            >
              <lfx-icon
                name="arrow-right-from-bracket"
                type="light"
                :size="16"
              />
              Sign out
            </button>
          </div>
        </div>
      </div>
    </div>
  </header>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue';
import LfxButton from '~/components/uikit/button/button.vue';
import LfxIcon from '~/components/uikit/icon/icon.vue';
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
const moreRef = ref<HTMLElement | null>(null);

const userOpen = ref(false);
const userRef = ref<HTMLElement | null>(null);

function onDocClick(e: MouseEvent) {
  if (moreRef.value && !moreRef.value.contains(e.target as Node)) {
    moreOpen.value = false;
  }
  if (userRef.value && !userRef.value.contains(e.target as Node)) {
    userOpen.value = false;
  }
}

onMounted(() => document.addEventListener('click', onDocClick));
onUnmounted(() => document.removeEventListener('click', onDocClick));
</script>

<script lang="ts">
export default {
  name: 'CrowdfundingHeader',
};
</script>
