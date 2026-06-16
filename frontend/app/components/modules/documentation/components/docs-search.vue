<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div
    ref="containerRef"
    class="relative"
  >
    <lfx-input
      v-model="query"
      placeholder="Search documentation…"
      @focus="onFocus"
    >
      <template #prefix>
        <lfx-icon
          name="magnifying-glass"
          type="light"
          :size="14"
          class="text-neutral-400"
        />
      </template>
      <template
        v-if="query"
        #suffix
      >
        <button
          type="button"
          class="text-neutral-400 hover:text-neutral-600"
          aria-label="Clear search"
          @click="clear"
        >
          <lfx-icon
            name="xmark"
            type="light"
            :size="12"
          />
        </button>
      </template>
    </lfx-input>

    <!-- Results panel -->
    <div
      v-if="isOpen && debouncedQuery.length >= 2"
      class="absolute left-0 right-0 top-full z-50 mt-1.5 overflow-hidden rounded-xl border border-neutral-200 bg-white shadow-lg"
    >
      <div
        v-if="isIndexLoading"
        class="px-4 py-3 text-sm text-neutral-400"
      >
        Loading…
      </div>
      <div
        v-else-if="results.length === 0"
        class="px-4 py-3 text-sm text-neutral-400"
      >
        No results for "<span class="text-neutral-700">{{ debouncedQuery }}</span
        >"
      </div>
      <ul
        v-else
        class="max-h-80 divide-y divide-neutral-100 overflow-y-auto"
      >
        <li
          v-for="result in results"
          :key="result.slug"
        >
          <NuxtLink
            :to="`/docs/${result.slug}`"
            class="flex flex-col gap-0.5 px-4 py-3 transition-colors hover:bg-brand-50"
            @click="clear"
          >
            <span class="text-sm font-medium text-neutral-900">{{ result.title }}</span>
            <span
              v-if="result.description"
              class="line-clamp-1 text-xs text-neutral-500"
            >
              {{ result.description }}
            </span>
          </NuxtLink>
        </li>
      </ul>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, useTemplateRef } from 'vue';
import { onClickOutside, refDebounced } from '@vueuse/core';
import MiniSearch from 'minisearch';
import LfxInput from '~/components/uikit/input/input.vue';
import LfxIcon from '~/components/uikit/icon/icon.vue';
import type { DocSearchDocument, DocSearchResult } from '#shared/types/documentation.types';

const query = ref('');
const debouncedQuery = refDebounced(query, 250);
const isOpen = ref(false);
const isIndexLoading = ref(false);
const results = ref<DocSearchResult[]>([]);

let searchIndex: MiniSearch<DocSearchDocument> | null = null;

const containerRef = useTemplateRef<HTMLDivElement>('containerRef');
onClickOutside(containerRef, () => {
  isOpen.value = false;
});

async function ensureIndex(): Promise<void> {
  if (searchIndex) return;
  isIndexLoading.value = true;
  try {
    const docs = await $fetch<DocSearchDocument[]>('/assets/docs/search-index.json');
    searchIndex = new MiniSearch<DocSearchDocument>({
      fields: ['title', 'description', 'content'],
      storeFields: ['slug', 'title', 'description'],
      idField: 'slug',
    });
    searchIndex.addAll(docs);
  } catch {
    // Search index unavailable — leave searchIndex null so results stay empty
  } finally {
    isIndexLoading.value = false;
  }
}

function onFocus(): void {
  isOpen.value = true;
  if (debouncedQuery.value.length >= 2) void ensureIndex();
}

watch(debouncedQuery, async (q) => {
  if (q.length < 2) {
    results.value = [];
    return;
  }
  await ensureIndex();
  if (!searchIndex) return;
  results.value = searchIndex
    .search(q, { boost: { title: 3, description: 2 }, fuzzy: 0.2, prefix: true })
    .slice(0, 8) as DocSearchResult[];
});

function clear(): void {
  query.value = '';
  results.value = [];
  isOpen.value = false;
}
</script>

<script lang="ts">
export default {
  name: 'DocsSearch',
};
</script>
