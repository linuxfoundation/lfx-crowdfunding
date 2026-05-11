<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <footer class="border-t border-neutral-200 py-10 bg-white">
    <div class="container">
      <div class="flex justify-between pb-10 md:pb-16 gap-x-10 gap-y-15 flex-col xl:flex-row">
        <!-- Brand -->
        <div class="max-w-100">
          <img
            src="~/assets/images/logo.svg"
            alt="LFX Logo"
            class="h-6"
            loading="lazy"
            width="176"
            height="24"
          />
          <p class="pt-3 text-body-2 text-neutral-500">
            LFX Crowdfunding helps open source projects, mentorships, and events secure the funding they need to thrive.
          </p>
          <div class="pt-10 md:pt-11 lg:pt-15 flex items-center gap-3">
            <a
              href="https://github.com/linuxfoundation"
              target="_blank"
              rel="noopener noreferrer"
              aria-label="Linux Foundation on GitHub"
            >
              <lfx-icon-button
                size="small"
                type="secondary"
              >
                <lfx-icon
                  name="github"
                  type="brands"
                />
              </lfx-icon-button>
            </a>
            <a
              href="https://community.lfx.dev"
              target="_blank"
              rel="noopener noreferrer"
            >
              <lfx-button
                type="secondary"
                button-style="pill"
                size="small"
              >
                <lfx-icon name="messages" />
                Join community
              </lfx-button>
            </a>
          </div>
        </div>

        <!-- Nav sections -->
        <div class="flex gap-x-10 gap-y-8 lg:gap-x-20 flex-col lg:flex-row">
          <section
            v-for="section of lfxFooterMenu"
            :key="section.title"
          >
            <p class="text-xs leading-4 font-semibold text-neutral-400 pb-2">
              {{ section.title }}
            </p>
            <nav class="flex flex-col">
              <a
                v-for="link of section.links"
                :key="link.name"
                :href="link.link"
                target="_blank"
                rel="noopener noreferrer"
                class="text-sm leading-7 hover:underline whitespace-nowrap"
              >
                {{ link.name }}
              </a>
            </nav>
          </section>
        </div>
      </div>

      <!-- LFX web component footer (copyright, cookie policy, etc.) -->
      <ClientOnly>
        <lfx-footer
          cookie-tracking
          class="footer !p-0 !text-left max-w-190"
        />
      </ClientOnly>
    </div>
  </footer>
</template>

<script setup lang="ts">
import { nextTick } from 'vue';
import LfxIconButton from '~/components/uikit/icon-button/icon-button.vue';
import LfxButton from '~/components/uikit/button/button.vue';
import LfxIcon from '~/components/uikit/icon/icon.vue';
import { lfxFooterMenu } from '~/config/menu/footer';

if (import.meta.client) {
  await import('@linuxfoundation/lfx-ui-core');
}

onMounted(async () => {
  await nextTick();
  // Align lfx-footer web component content to the left (no public API for this)
  const footer = document.querySelector('.footer') as HTMLElement;
  const shadowRoot = footer?.shadowRoot;
  if (shadowRoot) {
    const container = shadowRoot.querySelector('.footer-content');
    if (container) {
      (container as HTMLElement).style.textAlign = 'left';
    }
  }
});
</script>

<script lang="ts">
export default {
  name: 'CrowdfundingFooter',
};
</script>
