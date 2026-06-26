<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="pt-16 md:pb-30 pb-20">
    <div
      class="bg-gradient-to-b from-neutral-50 to-accent-100 rounded-xl md:p-16 p-6 flex md:items-end items-start justify-between gap-10 md:flex-row flex-col"
    >
      <div class="flex-1 flex flex-col gap-3 min-w-0">
        <h2 class="text-3xl font-secondary font-light leading-11 text-neutral-900 md:whitespace-nowrap">
          Ready to invest in open source?
        </h2>
        <p class="text-base font-normal leading-6 text-neutral-900">
          Our team will help you design a giving strategy that aligns with your business priorities and maximizes your
          impact.
        </p>
      </div>
      <div class="flex md:items-center items-start md:flex-row flex-col gap-4 md:w-auto w-full">
        <NuxtLink
          :to="AppRoute.Initiatives"
          class="md:w-auto w-full md:shrink-0"
        >
          <lfx-button
            icon="folder-heart"
            label="Browse initiatives"
            type="outline"
            button-style="pill"
            class="w-full justify-center"
          />
        </NuxtLink>
        <lfx-button
          icon="envelope"
          label="Contact our team"
          type="primary"
          button-style="pill"
          class="md:w-auto w-full md:shrink-0 justify-center"
          @click="openIntercom"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import LfxButton from '~/components/uikit/button/button.vue';
import { AppRoute } from '~/config/routes';
import { useIntercom } from '~/composables/useIntercom';

const { show } = useIntercom();

function openIntercom() {
  show();
}
</script>

<script lang="ts">
export default {
  name: 'ForCompaniesCta',
};
</script>
