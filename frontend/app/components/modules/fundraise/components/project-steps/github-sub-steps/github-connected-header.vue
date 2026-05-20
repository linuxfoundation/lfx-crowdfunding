<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="flex items-center gap-3 p-6">
    <div class="relative shrink-0 size-10">
      <div class="size-10 rounded-full bg-white border border-neutral-200 shadow-sm flex items-center justify-center">
        <lfx-icon
          name="github"
          type="brands"
          :size="22"
          class="text-neutral-900"
        />
      </div>
      <div
        class="absolute -bottom-0.5 -right-0.5 size-4 rounded-full bg-positive-500 border-2 border-white flex items-center justify-center"
      >
        <lfx-icon
          name="check"
          type="solid"
          :size="8"
          class="text-white"
        />
      </div>
    </div>
    <div class="flex-1 flex flex-col gap-0.5">
      <p class="text-sm font-semibold text-neutral-900">GitHub Connected</p>
      <p class="text-xs text-neutral-500 leading-4">
        Signed in as <span class="font-medium text-neutral-900">@{{ login }}</span>
      </p>
    </div>

    <lfx-button
      label="Disconnect"
      type="ghost"
      button-style="pill"
      @click="$emit('disconnect')"
    />
  </div>
</template>

<script setup lang="ts">
import LfxIcon from '~/components/uikit/icon/icon.vue';
import LfxButton from '~/components/uikit/button/button.vue';

defineProps<{
  login: string;
}>();

defineEmits<{
  (e: 'disconnect'): void;
}>();
</script>

<script lang="ts">
export default {
  name: 'GithubConnectedHeader',
};
</script>
