<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <lfx-drawer
    v-model="open"
    position="left"
    width="350px"
    :hide-close-button="true"
  >
    <template #default="{ close }">
      <div class="flex h-full flex-col justify-between p-5">
        <!-- Nav groups + close button -->
        <div class="flex flex-col gap-4">
          <!-- Group 1: main nav + close button -->
          <div class="relative flex flex-col gap-3">
            <NuxtLink
              v-for="item in regularMenuItems"
              :key="item.label"
              :to="item.to!"
              class="inline-flex items-center gap-2 rounded-full px-3 py-2 text-sm font-medium text-neutral-500 hover:bg-neutral-50 hover:text-neutral-700"
              active-class="!bg-neutral-100 !text-neutral-900"
              @click="close"
            >
              <lfx-icon
                :name="item.icon"
                type="light"
                :size="16"
              />
              {{ item.label }}
            </NuxtLink>

            <!-- Close button -->
            <lfx-icon-button
              type="outline"
              icon="xmark"
              icon-type="light"
              :icon-size="18"
              aria-label="Close menu"
              class="absolute right-0 top-0"
              @click="close"
            />
          </div>

          <!-- Divider -->
          <div class="h-px w-full bg-neutral-200" />

          <!-- Group 2: more items -->
          <div
            v-if="moreItem"
            class="flex flex-col gap-3"
          >
            <NuxtLink
              v-for="child in moreItem.children"
              :key="child.label"
              :to="child.to"
              class="inline-flex items-center gap-2 rounded-full px-3 py-2 text-sm font-medium text-neutral-500 hover:bg-neutral-50 hover:text-neutral-700"
              active-class="!bg-neutral-100 !text-neutral-900"
              @click="close"
            >
              <lfx-icon
                :name="child.icon"
                type="light"
                :size="16"
              />
              {{ child.label }}
            </NuxtLink>
          </div>
        </div>

        <!-- Bottom: CTAs -->
        <div class="flex flex-col gap-3.5">
          <start-fundraise-button class="w-full justify-center" />
        </div>
      </div>
    </template>
  </lfx-drawer>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import LfxDrawer from '~/components/uikit/drawer/drawer.vue';
import StartFundraiseButton from '~/components/shared/components/start-fundraise-button.vue';
import LfxIcon from '~/components/uikit/icon/icon.vue';
import LfxIconButton from '~/components/uikit/icon-button/icon-button.vue';
import { lfxHeaderMenu } from '~/config/menu/header';

const open = defineModel<boolean>({ required: true });

const regularMenuItems = computed(() => lfxHeaderMenu.filter((i) => !i.children));
const moreItem = computed(() => lfxHeaderMenu.find((i) => !!i.children));
</script>

<script lang="ts">
export default {
  name: 'CrowdfundingMobileMenu',
};
</script>
