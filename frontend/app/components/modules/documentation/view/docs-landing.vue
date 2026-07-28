<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="bg-white">
    <div class="container">
      <div class="md:pt-18 pt-10">
        <!-- Hero -->
        <div class="pb-16 flex flex-col gap-5">
          <div class="flex items-center gap-2">
            <lfx-icon
              name="book-open"
              type="light"
              :size="20"
              class="text-accent-800"
            />
            <span class="text-lg font-medium leading-7 text-accent-800">Documentation</span>
          </div>

          <h1 class="md:text-5xl text-4xl font-secondary font-light leading-tight text-black">LFX Crowdfunding Help</h1>

          <p class="text-base font-normal leading-6 text-neutral-900">
            Everything you need to know about donating to open source projects, creating fundraising initiatives, and
            managing your giving on LFX Crowdfunding.
          </p>

          <div class="w-full">
            <docs-search />
          </div>
        </div>

        <!-- Section grid -->
        <div class="pb-25">
          <template v-if="isLoading">
            <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
              <lfx-skeleton
                v-for="n in 4"
                :key="n"
                height="9rem"
                custom-class="rounded-2xl"
              />
            </div>
          </template>

          <template v-else-if="sections.length">
            <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
              <NuxtLink
                v-for="section in sections"
                :key="section.slug"
                :to="`/docs/${section.slug}`"
                class="flex flex-col gap-2 rounded-2xl border border-neutral-200 bg-white p-6 transition-shadow duration-200 hover:shadow-lg"
              >
                <div class="flex h-9 w-9 items-center justify-center rounded-lg bg-brand-50 text-brand-600">
                  <lfx-icon
                    :name="getSectionIcon(section.slug)"
                    type="light"
                    :size="16"
                  />
                </div>
                <h2 class="text-base font-semibold text-neutral-900">
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
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import DocsSearch from '../components/docs-search.vue';
import { getSectionIcon } from '../section-icons';
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
