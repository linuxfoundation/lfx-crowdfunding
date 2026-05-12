<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="flex flex-col justify-between border border-neutral-200 rounded-2xl p-6 h-[400px]">
    <div class="flex flex-col gap-6">
      <lfx-skeleton
        width="4rem"
        height="4rem"
        :rounded="true"
      />
      <div class="flex flex-col gap-4">
        <lfx-skeleton
          width="6rem"
          height="1rem"
          :rounded="true"
        />
        <div class="flex flex-col gap-1">
          <lfx-skeleton height="1.25rem" />
          <lfx-skeleton
            width="75%"
            height="1rem"
          />
        </div>
        <div class="flex gap-2">
          <lfx-skeleton
            width="4rem"
            height="1.25rem"
            :rounded="true"
          />
          <lfx-skeleton
            width="5rem"
            height="1.25rem"
            :rounded="true"
          />
        </div>
      </div>
    </div>
    <div class="flex flex-col gap-2">
      <lfx-skeleton height="1rem" />
      <lfx-skeleton
        height="0.25rem"
        :rounded="true"
      />
      <lfx-skeleton
        width="50%"
        height="1rem"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import LfxSkeleton from '~/components/uikit/skeleton/skeleton.vue';
</script>

<script lang="ts">
export default {
  name: 'InitiativeCardLoading',
};
</script>
