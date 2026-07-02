<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="flex flex-col gap-5">
    <div class="flex flex-col gap-1">
      <h2 class="text-base font-semibold text-neutral-900">Tiers</h2>
      <p class="text-xs text-neutral-900 leading-4">
        Choose the tiers you'd like to offer, set a donation amount for each, and optionally list the benefits sponsors
        will receive.
      </p>
    </div>

    <div class="flex flex-col gap-3">
      <fundraise-donation-tier-card
        v-for="(tier, index) in modelValue"
        :key="tier.name"
        :model-value="tier"
        @update:model-value="updateTier(index, $event)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import FundraiseDonationTierCard from './fundraise-donation-tier-card.vue';
import type { SponsorshipTier } from '~/types/fundraise.types';

const props = defineProps<{
  modelValue: SponsorshipTier[];
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: SponsorshipTier[]): void;
}>();

const updateTier = (index: number, tier: SponsorshipTier) => {
  emit(
    'update:modelValue',
    props.modelValue.map((t, i) => (i === index ? tier : t)),
  );
};
</script>

<script lang="ts">
export default {
  name: 'FundraiseDonationTiersSection',
};
</script>
