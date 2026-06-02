<!--
Copyright The Linux Foundation and each contributor to LFX.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="flex flex-col gap-6">
    <!-- Sponsorship tiers -->
    <!-- <donate-sponsorship-tiers
      :selected-tier-id="form.tierId"
      @select="selectTier"
    /> -->

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
import type { DonateAmountForm } from '#shared/types/donate.types';
import LfxInput from '~/components/uikit/input/input.vue';
// import DonateSponsorshipTiers from './donate-sponsorship-tiers.vue'; Hiding sponsorship tiers for now

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

// const selectTier = (tier: SponsorshipTier) => {
//   emit('update:modelValue', {
//     tierId: tier.id,
//     tierName: tier.name,
//     customAmountCents: null,
//     amountCents: tier.amountCents,
//   });
// };

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
</script>

<script lang="ts">
export default {
  name: 'DonateStepAmount',
};
</script>
