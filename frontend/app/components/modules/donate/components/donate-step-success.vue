<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="flex flex-col items-center gap-6 py-10 text-center">
    <!-- Heart icon -->
    <div class="flex size-16 items-center justify-center rounded-full bg-accent-500">
      <lfx-icon
        name="heart"
        type="solid"
        :size="28"
        class="text-white"
      />
    </div>

    <!-- Title + message -->
    <div class="flex flex-col gap-2">
      <h2 class="text-2xl font-semibold text-neutral-900">Thank you for your donation!</h2>
      <p class="text-sm text-neutral-600">
        Your {{ formattedAmount }} donation to
        <span class="font-semibold text-accent-600">{{ initiativeName }}</span>
        has been processed.
      </p>
    </div>

    <!-- Tier badge -->
    <span
      v-if="tierName"
      class="rounded-full px-3 py-1 text-xs font-semibold"
      :class="tierBadgeClass"
    >
      {{ tierName }} Sponsor
    </span>

    <!-- Share card -->
    <div class="w-full rounded-xl border border-neutral-200 p-5 flex flex-col items-center gap-4">
      <p class="text-sm font-medium text-neutral-700">Share your contribution</p>
      <div class="flex gap-3">
        <lfx-button
          type="outline"
          label="Post on X"
          icon="x-twitter"
          icon-type="brands"
          @click="shareOnX()"
        />
        <lfx-button
          type="outline"
          label="Share on LinkedIn"
          icon="linkedin-in"
          icon-type="brands"
          @click="shareOnLinkedIn()"
        />
      </div>
      <p class="text-xs text-neutral-400">Help {{ initiativeName }} reach its goal by spreading the word</p>
    </div>

    <!-- Manage donations link -->
    <a
      :href="selfServeLinks"
      target="_blank"
      rel="noopener noreferrer"
      class="flex items-center gap-1.5 text-sm font-medium text-accent-600 hover:text-accent-700 transition-colors"
    >
      Manage your Donations
      <lfx-icon
        name="arrow-up-right-from-square"
        type="light"
        :size="13"
      />
    </a>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import LfxIcon from '~/components/uikit/icon/icon.vue';
import LfxButton from '~/components/uikit/button/button.vue';

const TIER_BADGE_CLASSES: Record<string, string> = {
  bronze: 'bg-warning-100 text-warning-800',
  silver: 'bg-neutral-200 text-neutral-600',
  gold: 'bg-warning-200 text-warning-700',
  platinum: 'bg-neutral-100 text-neutral-500',
};

const props = defineProps<{
  amountCents: number;
  tierName: string | null;
  initiativeName: string;
}>();

const formattedAmount = computed(() => {
  const dollars = props.amountCents / 100;
  return dollars >= 1_000 ? `$${(dollars / 1_000).toLocaleString()}K` : `$${dollars.toLocaleString()}`;
});

const tierBadgeClass = computed(() => {
  if (!props.tierName) return '';
  return TIER_BADGE_CLASSES[props.tierName.toLowerCase()] ?? 'bg-neutral-100 text-neutral-600';
});

const {
  public: { selfServeUrl },
} = useRuntimeConfig();

const selfServeLinks = computed(() => `${selfServeUrl}/crowdfunding/donations`);

const shareOnX = () => {
  const text = encodeURIComponent(`I just donated to ${props.initiativeName} on LFX Crowdfunding!`);
  window.open(`https://twitter.com/intent/tweet?text=${text}`, '_blank');
};

const shareOnLinkedIn = () => {
  const url = encodeURIComponent(window.location.href);
  window.open(`https://www.linkedin.com/sharing/share-offsite/?url=${url}`, '_blank');
};
</script>

<script lang="ts">
export default {
  name: 'DonateStepSuccess',
};
</script>
