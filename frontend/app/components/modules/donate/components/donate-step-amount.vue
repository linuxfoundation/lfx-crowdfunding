<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="flex flex-col gap-6">
    <!-- Sponsorship tiers -->
    <div>
      <p class="text-sm font-medium text-neutral-700 mb-4">Select a sponsorship tier</p>
      <div class="grid grid-cols-4 rounded-xl border border-neutral-200 border-solid">
        <div
          v-for="tier in SPONSORSHIP_TIERS"
          :key="tier.id"
          class="flex flex-col gap-3 p-4 text-left transition-colors cursor-pointer border-neutral-200 border-solid first:rounded-l-xl last:rounded-r-xl border-r last:border-r-0"
          :class="
            form.tierId === tier.id
              ? 'border-accent-500 bg-accent-50'
              : 'border-neutral-200 bg-white hover:border-neutral-300'
          "
          @click="selectTier(tier)"
        >
          <div class="flex items-center justify-between">
            <span
              class="rounded-full px-2.5 py-0.5 text-xs font-semibold"
              :class="TIER_BADGE_CLASSES[tier.id]"
            >
              {{ tier.name }}
            </span>
            <lfx-radio
              :model-value="form.tierId ?? ''"
              :value="tier.id"
              @update:model-value="selectTier(tier)"
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

    <!-- Custom amount -->
    <div>
      <p class="text-sm font-medium text-neutral-700 mb-4">Enter a custom amount</p>

      <div class="flex flex-wrap items-center gap-2 mb-4">
        <button
          v-for="amount in QUICK_AMOUNTS"
          :key="amount"
          type="button"
          class="rounded-full border px-4 py-1.5 text-sm font-medium transition-colors"
          :class="
            isQuickAmountSelected(amount)
              ? 'border-accent-500 bg-accent-50 text-accent-600'
              : 'border-neutral-300 text-neutral-700 hover:border-neutral-400'
          "
          @click="selectQuickAmount(amount)"
        >
          ${{ amount }}
        </button>
      </div>

      <lfx-input
        :model-value="customAmountDisplay"
        placeholder="Enter amount"
        type="number"
        @update:model-value="onCustomAmountInput"
      >
        <template #prefix>$</template>
      </lfx-input>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { DonateAmountForm, SponsorshipTier } from '#shared/types/donate.types';
import LfxRadio from '~/components/uikit/radio/radio.vue';
import LfxIcon from '~/components/uikit/icon/icon.vue';
import LfxInput from '~/components/uikit/input/input.vue';

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

const QUICK_AMOUNTS = [5, 10, 25, 50, 150, 200];

const props = defineProps<{
  modelValue: DonateAmountForm;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: DonateAmountForm): void;
}>();

const form = computed(() => props.modelValue);

const customAmountDisplay = computed(() => {
  if (form.value.tierId !== null) return '';
  return form.value.customAmountCents ? String(form.value.customAmountCents / 100) : '';
});

const selectTier = (tier: SponsorshipTier) => {
  emit('update:modelValue', {
    tierId: tier.id,
    tierName: tier.name,
    customAmountCents: null,
    amountCents: tier.amountCents,
  });
};

const selectQuickAmount = (dollars: number) => {
  const cents = dollars * 100;
  emit('update:modelValue', {
    tierId: null,
    tierName: null,
    customAmountCents: cents,
    amountCents: cents,
  });
};

const isQuickAmountSelected = (dollars: number) =>
  form.value.tierId === null && form.value.customAmountCents === dollars * 100;

const onCustomAmountInput = (val: string | number) => {
  const dollars = parseFloat(String(val));
  if (isNaN(dollars) || dollars <= 0) {
    emit('update:modelValue', { ...form.value, tierId: null, tierName: null, customAmountCents: null, amountCents: 0 });
    return;
  }
  const cents = Math.round(dollars * 100);
  emit('update:modelValue', {
    tierId: null,
    tierName: null,
    customAmountCents: cents,
    amountCents: cents,
  });
};

const formatTierAmount = (cents: number): string => {
  const dollars = cents / 100;
  if (dollars >= 1_000) return `${(dollars / 1_000).toLocaleString()}K`;
  return dollars.toLocaleString();
};
</script>

<script lang="ts">
export default {
  name: 'DonateStepAmount',
};
</script>
