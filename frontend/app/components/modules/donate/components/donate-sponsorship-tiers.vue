<!--
Copyright The Linux Foundation and each contributor to LFX.
SPDX-License-Identifier: MIT
-->
<template>
  <div>
    <p class="text-sm font-medium text-neutral-700 mb-4">Select a sponsorship tier</p>
    <div class="grid grid-cols-1 sm:grid-cols-4 rounded-xl border border-neutral-200 border-solid">
      <div
        v-for="tier in SPONSORSHIP_TIERS"
        :key="tier.id"
        role="button"
        tabindex="0"
        class="flex flex-col gap-3 p-4 text-left transition-colors cursor-pointer border-neutral-200 border-solid first:rounded-t-xl last:rounded-b-xl sm:first:rounded-t-none sm:last:rounded-b-none sm:first:rounded-l-xl sm:last:rounded-r-xl border-b last:border-b-0 sm:border-b-0 sm:border-r sm:last:border-r-0"
        :class="
          props.selectedTierId === tier.id
            ? 'border-accent-500 bg-accent-50'
            : 'border-neutral-200 bg-white hover:border-neutral-300'
        "
        @click="emit('select', tier)"
        @keydown.enter="emit('select', tier)"
        @keydown.space.prevent="emit('select', tier)"
      >
        <div class="flex items-center justify-between">
          <span
            class="rounded-full px-2.5 py-0.5 text-xs font-semibold"
            :class="TIER_BADGE_CLASSES[tier.id]"
          >
            {{ tier.name }}
          </span>
          <lfx-radio
            :model-value="props.selectedTierId ?? ''"
            :value="tier.id"
            @update:model-value="emit('select', tier)"
          />
        </div>

        <p class="text-2xl font-bold text-neutral-900">${{ formatTierAmount(tier.amountCents) }}</p>

        <ul class="flex flex-col gap-1.5">
          <li
            v-for="benefit in tier.benefits"
            :key="benefit"
            class="flex items-start gap-2 text-xs text-neutral-700"
          >
            <lfx-icon
              name="check"
              type="solid"
              :size="10"
              class="mt-0.5 shrink-0 text-positive-600"
            />
            <span>{{ benefit }}</span>
          </li>
        </ul>
      </div>
    </div>
  </div>

  <!-- OR divider -->
  <div class="flex items-center gap-4">
    <hr class="flex-1 border-neutral-200" />
    <span class="text-xs font-medium text-neutral-400">OR</span>
    <hr class="flex-1 border-neutral-200" />
  </div>
</template>

<script setup lang="ts">
import type { SponsorshipTier } from '#shared/types/donate.types';
import LfxRadio from '~/components/uikit/radio/radio.vue';
import LfxIcon from '~/components/uikit/icon/icon.vue';

const SPONSORSHIP_TIERS: SponsorshipTier[] = [
  {
    id: 'bronze',
    name: 'Bronze',
    amountCents: 50_000,
    benefits: ['Name on supporters page', 'Quarterly newsletter'],
  },
  {
    id: 'silver',
    name: 'Silver',
    amountCents: 500_000,
    benefits: ['Bronze benefits', 'Logo on project page', 'Early access to audit reports'],
  },
  {
    id: 'gold',
    name: 'Gold',
    amountCents: 2_500_000,
    benefits: ['Silver benefits', 'Logo on homepage', 'Direct access to audit team', 'Custom briefing'],
  },
  {
    id: 'platinum',
    name: 'Platinum',
    amountCents: 10_000_000,
    benefits: ['Gold benefits', 'Advisory board seat', 'Co-branded announcements', 'Executive briefing'],
  },
];

const TIER_BADGE_CLASSES: Record<string, string> = {
  bronze: 'bg-warning-100 text-warning-800',
  silver: 'bg-neutral-200 text-neutral-600',
  gold: 'bg-warning-200 text-warning-700',
  platinum: 'bg-neutral-100 text-neutral-500',
};

const props = defineProps<{
  selectedTierId: string | null;
}>();

const emit = defineEmits<{
  (e: 'select', tier: SponsorshipTier): void;
}>();

const formatTierAmount = (cents: number): string => {
  const dollars = cents / 100;
  if (dollars >= 1_000) return `${(dollars / 1_000).toLocaleString()}K`;
  return dollars.toLocaleString();
};
</script>

<script lang="ts">
export default {
  name: 'DonateSponsorshipTiers',
};
</script>
