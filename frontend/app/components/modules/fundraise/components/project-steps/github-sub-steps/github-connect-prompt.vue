<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="flex flex-col items-center gap-5 py-10 px-8 text-center">
    <div class="size-12 rounded-full bg-white border border-neutral-200 shadow-sm flex items-center justify-center">
      <lfx-icon
        name="github"
        type="brands"
        :size="26"
        class="text-neutral-900"
      />
    </div>
    <div class="flex flex-col gap-2">
      <p class="text-sm font-semibold text-neutral-900">Authorize with GitHub</p>
      <p class="text-xs text-neutral-600 leading-4 max-w-sm">
        We'll request read-only access to your public repositories to link your initiative and display project health
        metrics.
      </p>
    </div>
    <lfx-button
      label="Connect with GitHub"
      type="dark"
      button-style="pill"
      icon="github"
      icon-type="brands"
      @click="$emit('connect')"
    />
  </div>
</template>

<script setup lang="ts">
import LfxIcon from '~/components/uikit/icon/icon.vue';
import LfxButton from '~/components/uikit/button/button.vue';

defineEmits<{
  (e: 'connect'): void;
}>();
</script>

<script lang="ts">
export default {
  name: 'GithubConnectPrompt',
};
</script>
