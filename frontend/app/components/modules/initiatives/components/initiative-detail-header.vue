<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="border-b border-neutral-200">
    <div class="container pt-16 pb-6 flex flex-col md:gap-16 gap-8">
      <!-- Initiative info row -->
      <div class="flex gap-6 items-start w-full">
        <!-- Logo + info -->
        <div class="flex flex-1 min-w-0 gap-6 items-start md:flex-row flex-col">
          <div class="flex items-center justify-between md:w-auto w-full">
            <lfx-avatar
              type="organization"
              size="xlarge"
              class="md:!size-40 !size-11 !rounded-2xl shrink-0"
              :src="initiative.logoUrl"
            />
            <div class="md:hidden flex items-center gap-3">
              <lfx-icon-button
                type="outline"
                icon="share-nodes"
              />
              <lfx-icon-button
                type="outline"
                icon="github"
                icon-type="brands"
              />
            </div>
          </div>

          <div class="flex flex-col h-full justify-between">
            <div class="flex flex-col md:gap-1 gap-4">
              <!-- Type badge -->
              <div
                class="flex items-center gap-2"
                :class="typeConfig.colorClass"
              >
                <lfx-icon
                  :name="typeConfig.icon"
                  type="light"
                  :size="16"
                />
                <span class="text-sm font-medium leading-5">{{ typeConfig.label }}</span>
              </div>

              <!-- Title -->
              <div class="flex gap-1 flex-col">
                <h1 class="font-secondary font-light md:text-3xl text-2xl md:leading-[44px] leading-9 text-black">
                  {{ initiative.name }}
                </h1>

                <!-- Description -->
                <p class="text-sm text-neutral-600 leading-5">
                  {{ initiative.description }}
                </p>
              </div>
            </div>

            <!-- Industry chips -->
            <div
              v-if="tags.length"
              class="flex flex-wrap gap-2 md:mt-9 mt-4"
            >
              <lfx-chip
                v-for="tag in tags"
                :key="tag"
                type="bordered"
                size="xsmall"
              >
                {{ tag }}
              </lfx-chip>
            </div>
          </div>
        </div>

        <!-- Action buttons -->
        <div class="md:flex hidden items-center gap-4 shrink-0">
          <lfx-button
            label="Share"
            type="ghost"
            icon="share-nodes"
            icon-position="left"
          />
          <lfx-button
            v-if="initiative.githubURL"
            label="GitHub"
            type="ghost"
            icon="github"
            icon-type="brands"
            icon-position="left"
          />
          <lfx-button
            label="Fund this initiative"
            type="ghost"
            icon="hand-heart"
            icon-position="left"
            class="!text-accent-500"
            @click="openDonateDrawer({ id: initiative.id, name: initiative.name, logoUrl: initiative.logoUrl })"
          />
        </div>
      </div>

      <!-- Tabs -->
      <div class="flex gap-4">
        <button
          v-for="tab in tabs"
          :key="tab.value"
          class="flex items-center gap-1.5 h-9 px-3 py-1 rounded-full text-sm transition-colors"
          :class="
            activeTab === tab.value
              ? 'bg-accent-100 text-neutral-900 font-semibold'
              : 'text-neutral-900 font-medium hover:bg-neutral-50'
          "
          @click="$emit('update:activeTab', tab.value)"
        >
          <lfx-icon
            :name="tab.icon"
            :type="activeTab === tab.value ? 'solid' : 'light'"
            :size="16"
            :class="activeTab === tab.value ? 'text-accent-500' : 'text-neutral-900'"
          />
          <span>{{ tab.label }}</span>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import {
  initiativeTypeConfigMap,
  defaultInitiativeTypeConfig,
} from '../../../shared/components/initiative-card/initiative-card.config';
import type { InitiativeDetail } from '#shared/types/initiative-detail.types';
import LfxAvatar from '~/components/uikit/avatar/avatar.vue';
import LfxIcon from '~/components/uikit/icon/icon.vue';
import LfxChip from '~/components/uikit/chip/chip.vue';
import LfxButton from '~/components/uikit/button/button.vue';
import LfxIconButton from '~/components/uikit/icon-button/icon-button.vue';
import { useDonateDrawerStore } from '~/components/modules/donate/store/donate-drawer.store';

const props = defineProps<{
  initiative: InitiativeDetail;
  activeTab?: string;
}>();

const { openDonateDrawer } = useDonateDrawerStore();

defineEmits<{ (e: 'update:activeTab', value: string): void }>();

const tabs = [
  { value: 'overview', label: 'Overview', icon: 'gauge-high' },
  { value: 'financials', label: 'Financials', icon: 'money-check-dollar' },
  { value: 'about', label: 'About', icon: 'memo' },
];

const typeConfig = computed(
  () => initiativeTypeConfigMap[props.initiative.initiativeType] ?? defaultInitiativeTypeConfig,
);

const tags = computed(() => {
  const industry = props.initiative.industry;
  if (!industry) return [];
  return industry
    .split(',')
    .map((t) => t.trim())
    .filter(Boolean);
});
</script>

<script lang="ts">
export default {
  name: 'InitiativeDetailHeader',
};
</script>
