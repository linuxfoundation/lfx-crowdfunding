<!--
Copyright The Linux Foundation and each contributor to LFX.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="flex flex-col gap-6">
    <!-- Sponsorship tiers -->
    <donate-sponsorship-tiers
      v-if="showSponsorshipTiers"
      :tiers="props.sponsorshipTiers ?? []"
      :selected-tier-id="form.tierId"
      @select="selectTier"
    />

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

    <!-- Donation type -->
    <div>
      <p class="text-sm font-medium text-neutral-700 mb-3">Donation frequency</p>
      <div class="flex items-center gap-6">
        <lfx-radio
          name="donationType"
          :model-value="form.donationType"
          value="one-time"
          @update:model-value="onDonationTypeChange"
        >
          One-time
        </lfx-radio>
        <lfx-radio
          name="donationType"
          :model-value="form.donationType"
          value="monthly"
          @update:model-value="onDonationTypeChange"
        >
          Monthly
        </lfx-radio>
      </div>
    </div>

    <!-- Category -->
    <div v-if="categoryOptions.length > 0">
      <p class="text-sm font-medium text-neutral-700 mb-3">Donation Allocation</p>
      <lfx-select
        :model-value="form.category ?? ''"
        placeholder="Select a category"
        @update:model-value="onCategoryChange"
      >
        <lfx-dropdown-item
          value=""
          label="All project needs"
        />
        <lfx-dropdown-item
          v-for="item in categoryOptions"
          :key="item.id"
          :value="item.name"
          :label="item.name"
        />
      </lfx-select>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import DonateSponsorshipTiers from './donate-sponsorship-tiers.vue';
import type { DonateAmountForm, DonationType, SponsorshipTier } from '#shared/types/donate.types';
import type { FundingGoal } from '#shared/types/initiative-detail.types';
import LfxInput from '~/components/uikit/input/input.vue';
import LfxRadio from '~/components/uikit/radio/radio.vue';
import LfxSelect from '~/components/uikit/select/select.vue';
import LfxDropdownItem from '~/components/uikit/dropdown/dropdown-item.vue';
import { isSponsorshipTiersEnabled } from '~/utils/feature-flags';

const QUICK_AMOUNTS = [5, 10, 25, 50, 150, 200];

const props = defineProps<{
  modelValue: DonateAmountForm;
  fundingGoals?: FundingGoal[];
  sponsorshipTiers?: SponsorshipTier[];
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: DonateAmountForm): void;
}>();

const form = computed(() => props.modelValue);

const categoryOptions = computed(() => props.fundingGoals ?? []);

const showSponsorshipTiers = computed(() => isSponsorshipTiersEnabled() && (props.sponsorshipTiers?.length ?? 0) > 0);

const customAmountDisplay = computed(() => {
  if (form.value.tierId !== null) return '';
  return form.value.customAmountCents ? String(form.value.customAmountCents / 100) : '';
});

const selectTier = (tier: SponsorshipTier) => {
  emit('update:modelValue', {
    ...form.value,
    tierId: tier.id,
    tierName: tier.name,
    customAmountCents: null,
    amountCents: tier.amountCents,
  });
};

const selectQuickAmount = (dollars: number) => {
  const cents = dollars * 100;
  emit('update:modelValue', {
    ...form.value,
    tierId: null,
    tierName: null,
    customAmountCents: cents,
    amountCents: cents,
  });
};

const isQuickAmountSelected = (dollars: number) =>
  form.value.tierId === null && form.value.customAmountCents === dollars * 100;

const onDonationTypeChange = (val: string | number | boolean) => {
  emit('update:modelValue', { ...form.value, donationType: val as DonationType });
};

const onCategoryChange = (val: string) => {
  emit('update:modelValue', { ...form.value, category: val || null });
};

const onCustomAmountInput = (val: string | number) => {
  const dollars = parseFloat(String(val));
  if (isNaN(dollars) || dollars <= 0) {
    emit('update:modelValue', { ...form.value, tierId: null, tierName: null, customAmountCents: null, amountCents: 0 });
    return;
  }
  const cents = Math.round(dollars * 100);
  emit('update:modelValue', {
    ...form.value,
    tierId: null,
    tierName: null,
    customAmountCents: cents,
    amountCents: cents,
  });
};
</script>

<script lang="ts">
export default {
  name: 'DonateStepAmount',
};
</script>
