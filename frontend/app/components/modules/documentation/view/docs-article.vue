<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="bg-white">
    <div class="container px-5 md:px-10">
      <div class="flex gap-12 pt-10 pb-25">
        <!-- Sidebar -->
        <aside class="hidden w-64 shrink-0 lg:block">
          <div class="sticky top-24">
            <NuxtLink
              to="/docs"
              class="mb-4 flex items-center gap-1.5 text-sm text-neutral-500 hover:text-neutral-700"
            >
              <lfx-icon
                name="arrow-left"
                type="light"
                :size="12"
              />
              All docs
            </NuxtLink>
            <div class="mb-5">
              <docs-search />
            </div>
            <docs-sidebar />
          </div>
        </aside>

        <!-- Main content -->
        <main class="min-w-0 flex-1">
          <!-- Breadcrumb -->
          <nav
            class="mb-6 flex items-center gap-1.5 text-sm text-neutral-500"
            aria-label="Breadcrumb"
          >
            <NuxtLink
              to="/docs"
              class="hover:text-neutral-700"
            >
              Docs
            </NuxtLink>

            <template v-if="isLoading">
              <lfx-icon
                name="chevron-right"
                type="light"
                :size="10"
              />
              <span class="inline-block h-4 w-32 animate-pulse rounded bg-neutral-200" />
            </template>

            <template v-else>
              <template
                v-for="crumb in breadcrumbs"
                :key="crumb.slug"
              >
                <lfx-icon
                  name="chevron-right"
                  type="light"
                  :size="10"
                />
                <NuxtLink
                  :to="`/docs/${crumb.slug}`"
                  class="hover:text-neutral-700"
                >
                  {{ crumb.title }}
                </NuxtLink>
              </template>

              <template v-if="!isError">
                <lfx-icon
                  name="chevron-right"
                  type="light"
                  :size="10"
                />
                <span class="text-neutral-900">{{ article?.title }}</span>
              </template>
            </template>
          </nav>

          <!-- Article skeleton -->
          <template v-if="isLoading">
            <lfx-skeleton
              height="2.25rem"
              custom-class="mb-3 max-w-sm rounded-lg"
            />
            <lfx-skeleton
              height="1rem"
              custom-class="mb-1.5 rounded"
            />
            <lfx-skeleton
              height="1rem"
              custom-class="mb-1.5 w-4/5 rounded"
            />
            <lfx-skeleton
              height="1rem"
              custom-class="mb-8 w-3/5 rounded"
            />
            <lfx-skeleton
              height="1.5rem"
              custom-class="mb-3 max-w-xs rounded-lg"
            />
            <lfx-skeleton
              v-for="n in 4"
              :key="n"
              height="1rem"
              custom-class="mb-1.5 rounded"
            />
          </template>

          <!-- Error state -->
          <template v-else-if="isError">
            <div class="rounded-xl border border-red-200 bg-red-50 p-8 text-center">
              <lfx-icon
                name="circle-exclamation"
                type="light"
                :size="32"
                class="mb-3 text-red-400"
              />
              <p class="font-semibold text-red-700">Page not found</p>
              <p class="mt-1 text-sm text-red-600">This documentation page doesn't exist or has been moved.</p>
              <NuxtLink
                to="/docs"
                class="mt-4 inline-flex items-center gap-1.5 text-sm font-medium text-brand-600 hover:text-brand-700"
              >
                <lfx-icon
                  name="arrow-left"
                  type="light"
                  :size="12"
                />
                Back to all docs
              </NuxtLink>
            </div>
          </template>

          <!-- Article content -->
          <template v-else-if="article">
            <h1 class="mb-2 text-2xl font-bold text-neutral-900">{{ article.title }}</h1>
            <p
              v-if="article.description"
              class="mb-6 text-base text-neutral-500"
            >
              {{ article.description }}
            </p>

            <!-- Body -->
            <!-- eslint-disable-next-line vue/no-v-html -->
            <div
              class="lfx-rich-text docs-article-body"
              v-html="article.bodyHtml"
            />

            <!-- Footer metadata -->
            <div
              v-if="article.lastUpdated"
              class="mt-10 border-t border-neutral-100 pt-6 text-xs text-neutral-400"
            >
              Last updated: {{ article.lastUpdated }}
            </div>
          </template>
        </main>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import DocsSidebar from '../components/docs-sidebar.vue';
import DocsSearch from '../components/docs-search.vue';
import LfxIcon from '~/components/uikit/icon/icon.vue';
import LfxSkeleton from '~/components/uikit/skeleton/skeleton.vue';
import { useDocumentation } from '~/composables/documentation/useDocumentation';
import { useDocumentationNav } from '~/composables/documentation/useDocumentationNav';

const url = useRequestURL();

const props = defineProps<{
  slug: string;
}>();

const { data: article, isLoading, isError } = useDocumentation(() => props.slug);
const { data: navData } = useDocumentationNav();

const seoTitle = computed(() =>
  article.value ? `${article.value.title} — LFX Crowdfunding Docs` : 'LFX Crowdfunding Docs',
);
const seoDescription = computed(() => article.value?.description ?? '');

useHead({
  title: seoTitle,
  link: [{ rel: 'canonical', href: computed(() => url.origin + url.pathname) }],
});
useSeoMeta({
  ogTitle: seoTitle,
  ogDescription: seoDescription,
  ogType: 'article',
  ogUrl: computed(() => url.origin + url.pathname),
  twitterCard: 'summary',
  twitterTitle: seoTitle,
  twitterDescription: seoDescription,
});

// Build intermediate breadcrumb crumbs for nested slugs (e.g. 'initiatives/browsing-initiatives')
const breadcrumbs = computed(() => {
  const parts = props.slug.split('/');
  if (parts.length < 2) return [];

  const sections = navData.value?.sections ?? [];
  const parentSlug = parts[0];
  const parent = sections.find((s) => s.slug === parentSlug);

  return parent ? [{ title: parent.title, slug: parentSlug }] : [];
});
</script>

<script lang="ts">
export default {
  name: 'DocsArticle',
};
</script>
