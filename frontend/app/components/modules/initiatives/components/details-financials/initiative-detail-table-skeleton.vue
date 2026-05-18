<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="flex flex-col gap-4">
    <div
      v-for="n in rows"
      :key="n"
      class="flex items-center gap-3 border-t border-neutral-200 pt-4"
    >
      <lfx-skeleton
        v-for="(col, i) in columns"
        :key="i"
        :rounded="col.rounded"
        :width="col.width"
        :height="col.height ?? '0.875rem'"
        :class="col.class"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import LfxSkeleton from '~/components/uikit/skeleton/skeleton.vue';

interface SkeletonColumn {
  width: string;
  height?: string;
  rounded?: boolean;
  class?: string;
}

withDefaults(defineProps<{ columns: SkeletonColumn[]; rows?: number }>(), { rows: 5 });
</script>

<script lang="ts">
export default {
  name: 'InitiativeDetailTableSkeleton',
};
</script>
