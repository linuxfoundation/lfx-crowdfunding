<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <footer class="bg-accent-900">
    <div class="container py-10">
      <!-- Top: brand + nav columns -->
      <div class="flex items-start justify-between gap-10">
        <!-- Brand -->
        <div class="flex flex-col gap-3">
          <NuxtLink :to="AppRoute.Home">
            <img
              src="~/assets/images/logo.png"
              alt="LFX Crowdfunding"
              class="h-6 brightness-0 invert"
              loading="lazy"
            />
          </NuxtLink>
          <p class="text-xs leading-4 text-neutral-400">
            Funding the open source software that powers the world. 0% platform fees.
          </p>
        </div>

        <!-- Nav sections -->
        <div class="flex gap-16">
          <section
            v-for="section in lfxFooterMenu"
            :key="section.title"
            class="flex flex-col gap-2"
          >
            <p class="text-xs font-semibold leading-4 text-neutral-500">
              {{ section.title }}
            </p>
            <nav class="flex flex-col">
              <template
                v-for="link in section.links"
                :key="link.name"
              >
                <template v-if="link.link">
                  <NuxtLink
                    v-if="link.link.startsWith('/')"
                    :to="link.link"
                    class="text-sm leading-7 text-white hover:underline"
                  >
                    {{ link.name }}
                  </NuxtLink>
                  <a
                    v-else
                    :href="link.link"
                    target="_blank"
                    rel="noopener noreferrer"
                    class="text-sm leading-7 text-white hover:underline"
                  >
                    {{ link.name }}
                  </a>
                </template>
                <template v-else>
                  <button
                    type="button"
                    class="text-sm leading-7 text-white hover:underline -ml-1"
                    @click="link.action"
                  >
                    {{ link.name }}
                  </button>
                </template>
              </template>
            </nav>
          </section>
        </div>
      </div>

      <!-- Divider -->
      <div class="my-8 h-px bg-white/10" />

      <!-- LFX footer web component (copyright, cookie prefs, legal links) -->
      <ClientOnly>
        <lfx-footer
          ref="lfxFooterRef"
          cookie-tracking
          class="lfx-footer-wc !p-0 !text-left"
        />
      </ClientOnly>
    </div>
  </footer>
</template>

<script setup lang="ts">
import { nextTick, ref } from 'vue';
import { lfxFooterMenu } from '~/config/menu/footer';
import { AppRoute } from '~/config/routes';

const lfxFooterRef = ref<HTMLElement | null>(null);

onMounted(async () => {
  await nextTick();
  const el = document.querySelector('.lfx-footer-wc') as HTMLElement;
  const shadowRoot = el?.shadowRoot;
  if (!shadowRoot) return;

  // Align content left (no public API for this)
  const content = shadowRoot.querySelector('.footer-content') as HTMLElement | null;
  if (content) content.style.textAlign = 'left';

  // Inject styles to match the dark footer design
  const style = document.createElement('style');
  style.textContent = `
    :host {
      background: transparent !important;
      color: #90a1b9 !important;
    }
    * {
      color: #90a1b9 !important;
      background: transparent !important;
    }
    a {
      text-decoration: underline !important;
    }
    a:hover {
      color: #cbd5e1 !important;
    }
    button {
      text-decoration: underline !important;
    }
    button:hover {
      color: #cbd5e1 !important;
    }
  `;
  shadowRoot.appendChild(style);
});
</script>

<script lang="ts">
export default {
  name: 'CrowdfundingFooter',
};
</script>
