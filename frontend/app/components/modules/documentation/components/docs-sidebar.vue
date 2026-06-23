<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <nav
    aria-label="Documentation sections"
    class="flex flex-col gap-1"
  >
    <div class="mb-3 px-3 text-xs font-semibold uppercase tracking-wider text-neutral-400">Documentation</div>

    <template v-if="isLoading">
      <lfx-skeleton
        v-for="n in 4"
        :key="n"
        height="2rem"
        custom-class="rounded-lg"
      />
    </template>

    <template v-else>
      <div
        v-for="section in sections"
        :key="section.slug"
      >
        <NuxtLink
          :to="`/docs/${section.slug}`"
          class="flex items-center gap-2.5 rounded-lg px-3 py-2 text-sm font-medium text-neutral-600 transition-colors hover:bg-neutral-50 hover:text-neutral-900"
          :class="{ '!bg-brand-50 !text-brand-700': isExactActive(section.slug) }"
        >
          <lfx-icon
            name="file-lines"
            type="light"
            :size="14"
            class="shrink-0 text-neutral-400"
          />
          {{ section.title }}
        </NuxtLink>

        <!-- Children — always visible when parent is active -->
        <div
          v-if="section.children.length && isParentActive(section.slug)"
          class="ml-4 mt-0.5 flex flex-col gap-0.5 border-l border-neutral-100 pl-3"
        >
          <NuxtLink
            v-for="child in section.children"
            :key="child.slug"
            :to="`/docs/${child.slug}`"
            class="rounded-md px-2 py-1.5 text-sm text-neutral-500 transition-colors hover:bg-neutral-50 hover:text-neutral-900"
            :class="{ '!text-brand-700 font-medium': isExactActive(child.slug) }"
          >
            {{ child.title }}
          </NuxtLink>
        </div>
      </div>
    </template>
  </nav>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useRoute } from 'nuxt/app';
import LfxSkeleton from '~/components/uikit/skeleton/skeleton.vue';
import LfxIcon from '~/components/uikit/icon/icon.vue';
import { useDocumentationNav } from '~/composables/documentation/useDocumentationNav';

const { data, isLoading } = useDocumentationNav();
const route = useRoute();

const sections = computed(() => data.value?.sections ?? []);

function isExactActive(slug: string): boolean {
  return route.path === `/docs/${slug}`;
}

function isParentActive(slug: string): boolean {
  return route.path.startsWith(`/docs/${slug}`);
}
</script>

<script lang="ts">
export default {
  name: 'DocsSidebar',
};
</script>
