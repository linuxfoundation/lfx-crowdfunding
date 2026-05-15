<template>
  <div class="flex items-center gap-8">
    <div
      v-for="badge in trustBadges"
      :key="badge.label"
      class="flex items-center gap-3"
    >
      <div
        class="size-10 rounded-full bg-white border border-neutral-200 shadow-sm flex items-center justify-center shrink-0"
      >
        <lfx-icon
          :name="badge.icon"
          type="light"
          :size="18"
        />
      </div>
      <span class="text-sm font-medium leading-5">{{ badge.label }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import LfxIcon from '~/components/uikit/icon/icon.vue';

const trustBadges = [
  { icon: 'circle-dollar', label: '0% fees' },
  { icon: 'building-columns', label: '501(c)(6) fiscal host' },
  { icon: 'eye', label: 'Full financial transparency' },
];
</script>
<script lang="ts">
export default {
  name: 'TrustBadge',
};
</script>
