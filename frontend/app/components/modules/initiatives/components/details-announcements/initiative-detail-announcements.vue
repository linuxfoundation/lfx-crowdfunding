<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="bg-white border border-neutral-200 rounded-2xl p-6 flex flex-col gap-8">
    <!-- Loading -->
    <template v-if="isLoading">
      <div
        v-for="n in 3"
        :key="n"
        class="flex gap-4 items-start"
      >
        <lfx-skeleton
          :rounded="true"
          width="0.625rem"
          height="0.625rem"
          class="mt-1 shrink-0"
        />
        <div class="flex flex-col gap-2 flex-1">
          <lfx-skeleton
            width="7rem"
            height="1rem"
          />
          <lfx-skeleton
            width="60%"
            height="1.25rem"
          />
          <lfx-skeleton
            width="100%"
            height="1rem"
          />
        </div>
      </div>
    </template>

    <!-- Empty -->
    <p
      v-else-if="!announcements.length"
      class="text-sm text-neutral-500"
    >
      No announcements yet.
    </p>

    <!-- Items -->
    <template v-else>
      <div
        v-for="(announcement, index) in announcements"
        :key="announcement.id"
        class="flex gap-4 items-start relative"
      >
        <!-- Timeline dot + line -->
        <div class="flex flex-col items-center shrink-0 mt-1">
          <lfx-icon
            name="circle-small"
            type="solid"
            :size="12"
            class="text-neutral-900"
          />
          <div
            v-if="index < announcements.length - 1"
            class="w-px flex-1 bg-neutral-200 mt-1"
            style="min-height: 1.5rem"
          />
        </div>

        <!-- Content -->
        <div class="flex flex-col gap-2 flex-1 min-w-0">
          <p class="text-xs text-neutral-500 leading-4">
            {{ formatShortDate(announcement.publishedAt) }}
          </p>
          <p class="text-base font-semibold text-neutral-900 leading-6">
            {{ announcement.title }}
          </p>
          <p class="text-sm text-neutral-600 leading-5">
            {{ announcement.body }}
          </p>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useInitiativeAnnouncements } from '~/composables/initiatives/useInitiativeAnnouncements';
import LfxSkeleton from '~/components/uikit/skeleton/skeleton.vue';
import LfxIcon from '~/components/uikit/icon/icon.vue';
import { formatShortDate } from '~/utils/date';

const props = defineProps<{ initiativeSlug: string }>();

const { data, isLoading } = useInitiativeAnnouncements(computed(() => props.initiativeSlug));

const announcements = computed(() => data.value?.data ?? []);
</script>

<script lang="ts">
export default {
  name: 'InitiativeDetailAnnouncements',
};
</script>
