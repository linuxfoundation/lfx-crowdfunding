<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="border-b border-neutral-200 sticky top-8 bg-white z-10">
    <div
      class="container pt-16 pb-6 flex flex-col transition-all ease-linear"
      :class="{ 'gap-8': isScrolled, 'md:gap-16 gap-8': !isScrolled }"
    >
      <!-- Initiative info row -->
      <div class="flex gap-6 items-start w-full">
        <!-- Logo + info -->
        <div class="flex flex-1 min-w-0 gap-6 items-start md:flex-row flex-col">
          <div class="flex items-center justify-between md:w-auto w-full">
            <lfx-avatar
              type="organization"
              class="!size-11 !rounded-xl"
              :class="{ 'md:!size-11': isScrolled, 'md:!size-40': !isScrolled }"
              :src="initiative.logoUrl"
            />
            <div class="md:hidden flex items-center gap-3">
              <lfx-icon-button
                type="outline"
                icon="share-nodes"
                @click="handleShare()"
              />
              <lfx-icon-button
                type="outline"
                icon="github"
                icon-type="brands"
              />
            </div>
          </div>

          <div class="flex flex-col h-full justify-between">
            <div
              class="flex flex-col"
              :class="{ 'gap-0': isScrolled, 'md:gap-1 gap-4': !isScrolled }"
            >
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
                <h1
                  class="font-secondary font-light text-black"
                  :class="{
                    'text-xl leading-8': isScrolled,
                    'md:text-3xl text-2xl md:leading-[44px] leading-9': !isScrolled,
                  }"
                >
                  {{ initiative.name }}
                </h1>

                <!-- Description -->
                <p
                  class="text-sm text-neutral-600 leading-5"
                  :class="{ hidden: isScrolled }"
                >
                  {{ plainDescription }}
                </p>
              </div>
            </div>

            <!-- Industry chips -->
            <div
              v-if="tags.length"
              class="flex flex-wrap gap-2 md:mt-9 mt-4"
              :class="{ hidden: isScrolled }"
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

            <!-- Funding disabled notice -->
            <p
              v-if="initiative.acceptFunding === false && !isScrolled"
              class="text-sm text-neutral-500 mt-2"
            >
              This initiative is not currently accepting donations.
            </p>
          </div>
        </div>

        <!-- Action buttons -->
        <div class="md:flex hidden items-center gap-4 shrink-0">
          <lfx-button
            label="Share"
            type="ghost"
            icon="share-nodes"
            icon-position="left"
            button-style="pill"
            @click="handleShare()"
          />
          <lfx-button
            v-if="initiative.githubURL"
            label="GitHub"
            type="ghost"
            icon="github"
            icon-type="brands"
            icon-position="left"
            button-style="pill"
          />
          <lfx-tooltip
            content="This initiative is not currently accepting donations"
            :disabled="initiative.acceptFunding !== false"
          >
            <span>
              <lfx-button
                label="Donate"
                type="primary"
                icon="hand-heart"
                icon-position="left"
                button-style="pill"
                :disabled="initiative.acceptFunding === false"
                @click="handleFundInitiative()"
              />
            </span>
          </lfx-tooltip>
        </div>
      </div>

      <!-- Tabs -->
      <lfx-tabs
        :model-value="activeTab"
        :tabs="tabs"
        tab-style="pill"
        @update:model-value="$emit('update:activeTab', $event)"
      />
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
import { useSanitize } from '~/composables/useSanitize';
import LfxIcon from '~/components/uikit/icon/icon.vue';
import LfxChip from '~/components/uikit/chip/chip.vue';
import LfxButton from '~/components/uikit/button/button.vue';
import LfxIconButton from '~/components/uikit/icon-button/icon-button.vue';
import LfxTabs from '~/components/uikit/tabs/tabs.vue';
import LfxTooltip from '~/components/uikit/tooltip/tooltip.vue';
import { useDonateDrawerStore } from '~/components/modules/donate/store/donate-drawer.store';
import { useShareModalStore } from '~/components/shared/components/share/store/share-modal.store';
import { useAuth } from '~/composables/useAuth';
import useScroll from '~/utils/scroll';

const props = defineProps<{
  initiative: InitiativeDetail;
  activeTab: string;
}>();

const { stripHtml } = useSanitize();
const plainDescription = computed(() => stripHtml(props.initiative.description ?? ''));

const { openDonateDrawer } = useDonateDrawerStore();
const { openShareModal } = useShareModalStore();
const { isAuthenticated, login } = useAuth();

function handleShare() {
  openShareModal({
    title: props.initiative.name,
    url: import.meta.client ? window.location.href : '',
  });
}

function handleFundInitiative() {
  if (!isAuthenticated.value) {
    login();
  } else {
    openDonateDrawer({
      id: props.initiative.id,
      name: props.initiative.name,
      logoUrl: props.initiative.logoUrl,
      fundingGoals: props.initiative.fundingGoals,
    });
  }
}

const { scrollTop } = useScroll();
const isScrolled = computed(() => scrollTop.value > 10);

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
