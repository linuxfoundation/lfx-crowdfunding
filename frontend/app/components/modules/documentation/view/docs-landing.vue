<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="bg-white">
    <!-- Hero -->
    <div class="border-b border-neutral-100 bg-neutral-50 py-12">
      <div class="container px-5 md:px-10">
        <div class="max-w-2xl">
          <div class="mb-2 flex items-center gap-2 text-sm text-neutral-500">
            <lfx-icon
              name="book-open"
              type="light"
              :size="14"
            />
            Documentation
          </div>
          <h1 class="text-3xl font-bold text-neutral-900">LFX Crowdfunding Help</h1>
          <p class="mt-3 text-base text-neutral-600">
            Everything you need to know about donating to open source projects, creating fundraising initiatives, and
            managing your giving on LFX Crowdfunding.
          </p>
          <div class="mt-5 max-w-md">
            <docs-search />
          </div>
        </div>
      </div>
    </div>

    <!-- Section grid -->
    <div class="container px-5 py-12 md:px-10">
      <template v-if="isLoading">
        <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <lfx-skeleton
            v-for="n in 4"
            :key="n"
            height="9rem"
            custom-class="rounded-xl"
          />
        </div>
      </template>

      <template v-else-if="sections.length">
        <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <NuxtLink
            v-for="section in sections"
            :key="section.slug"
            :to="`/docs/${section.slug}`"
            class="group flex flex-col gap-2 rounded-xl border border-neutral-200 bg-white p-6 transition-all hover:border-brand-300 hover:shadow-sm"
          >
            <div
              class="flex h-9 w-9 items-center justify-center rounded-lg bg-brand-50 text-brand-600 transition-colors group-hover:bg-brand-100"
            >
              <lfx-icon
                name="file-lines"
                type="light"
                :size="16"
              />
            </div>
            <h2 class="text-base font-semibold text-neutral-900 group-hover:text-brand-700">
              {{ section.title }}
            </h2>
            <p
              v-if="section.description"
              class="text-sm text-neutral-500 line-clamp-2"
            >
              {{ section.description }}
            </p>
          </NuxtLink>
        </div>
      </template>

      <p
        v-else
        class="text-sm text-neutral-500"
      >
        No documentation sections found.
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import DocsSearch from '../components/docs-search.vue';
import LfxIcon from '~/components/uikit/icon/icon.vue';
import LfxSkeleton from '~/components/uikit/skeleton/skeleton.vue';
import { useDocumentationNav } from '~/composables/documentation/useDocumentationNav';

const { data, isLoading } = useDocumentationNav();

const sections = computed(() => data.value?.sections ?? []);
</script>

<script lang="ts">
export default {
  name: 'DocsLanding',
};
</script>
