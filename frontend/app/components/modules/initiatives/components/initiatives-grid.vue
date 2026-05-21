<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <section class="container pb-16">
    <!-- Loading skeletons (initial load) -->
    <div
      v-if="isLoading"
      class="grid grid-cols-1 gap-8 sm:grid-cols-2 lg:grid-cols-3"
    >
      <initiative-card-loading
        v-for="n in 9"
        :key="n"
      />
    </div>

    <!-- Error -->
    <div
      v-else-if="error"
      class="flex items-center gap-2 text-negative-600"
    >
      <lfx-icon
        name="circle-exclamation"
        type="solid"
        :size="16"
      />
      <span class="text-body-1">Failed to load initiatives. Please try again.</span>
    </div>

    <!-- Empty state -->
    <div
      v-else-if="!initiatives.length"
      class="flex flex-col items-center justify-center gap-4 py-24 text-neutral-500"
    >
      <lfx-icon
        name="folder-open"
        type="light"
        :size="40"
      />
      <p class="text-base">No initiatives found.</p>
    </div>

    <!-- Initiative cards -->
    <template v-else>
      <div class="grid grid-cols-1 gap-8 sm:grid-cols-2 lg:grid-cols-3">
        <initiative-card
          v-for="initiative in initiatives"
          :key="initiative.id"
          :initiative="initiative"
        />
      </div>

      <!-- Bottom loading cards while fetching next page -->
      <div
        v-if="isFetchingNextPage"
        class="grid grid-cols-1 gap-8 mt-8 sm:grid-cols-2 lg:grid-cols-3"
      >
        <initiative-card-loading
          v-for="n in 3"
          :key="n"
        />
      </div>

      <!-- IntersectionObserver sentinel -->
      <div
        ref="sentinel"
        class="h-1"
      />
    </template>
  </section>
</template>

<script setup lang="ts">
import { ref, watch, onUnmounted } from 'vue';
import LfxIcon from '~/components/uikit/icon/icon.vue';
import InitiativeCard from '~/components/shared/components/initiative-card/initiative-card.vue';
import InitiativeCardLoading from '~/components/shared/components/initiative-card/initiative-card-loading.vue';
import type { InitiativeBase } from '~/types/initiative.types';

const props = defineProps<{
  initiatives: InitiativeBase[];
  isLoading: boolean;
  error: Error | null;
  isFetchingNextPage: boolean;
  hasNextPage: boolean;
}>();

const emit = defineEmits<{ (e: 'loadMore'): void }>();

const sentinel = ref<HTMLElement | null>(null);
let observer: IntersectionObserver | null = null;

const isSentinelVisible = (el: HTMLElement) => {
  const rect = el.getBoundingClientRect();
  const windowHeight = window.innerHeight || document.documentElement.clientHeight;
  return rect.top >= 0 && rect.bottom <= windowHeight;
};

const tryLoadMore = () => {
  if (props.hasNextPage && !props.isFetchingNextPage) {
    emit('loadMore');
  }
};

// Set up the observer only once the sentinel actually renders (it lives inside v-else,
// so it is absent from the DOM during the initial loading state).
watch(sentinel, (el) => {
  observer?.disconnect();
  observer = null;
  if (!el) return;

  observer = new IntersectionObserver(
    (entries) => {
      if (entries[0]?.isIntersecting) tryLoadMore();
    },
    { rootMargin: '200px' },
  );
  observer.observe(el);
});

watch(
  () => props.isFetchingNextPage,
  (fetching) => {
    if (!fetching && sentinel.value && isSentinelVisible(sentinel.value)) {
      tryLoadMore();
    }
  },
);

onUnmounted(() => {
  observer?.disconnect();
});
</script>

<script lang="ts">
export default {
  name: 'InitiativesGrid',
};
</script>
