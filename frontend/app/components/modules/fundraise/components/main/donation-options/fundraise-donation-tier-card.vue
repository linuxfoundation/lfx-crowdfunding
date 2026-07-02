<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="border border-neutral-200 rounded-xl p-5 flex flex-col gap-5">
    <lfx-checkbox
      :model-value="modelValue.enabled"
      @update:model-value="emit('update:modelValue', { ...modelValue, enabled: $event })"
    >
      <span class="flex items-center gap-2">
        <span
          class="inline-block size-3.5"
          :class="modelValue.name === 'platinum' ? 'tier-icon--diamond' : 'rounded-full'"
          :style="{ background: SPONSORSHIP_TIER_ICON_GRADIENT[modelValue.name] }"
        />
        <span class="text-sm font-semibold text-neutral-900">{{ SPONSORSHIP_TIER_LABEL[modelValue.name] }}</span>
      </span>
    </lfx-checkbox>

    <template v-if="modelValue.enabled">
      <div class="border-t border-neutral-200" />

      <!-- Sponsorship goal -->
      <div class="flex flex-col gap-3">
        <label class="text-xs font-medium text-neutral-900">
          Sponsorship Goal <span class="text-negative-500">*</span>
        </label>
        <lfx-input
          :model-value="modelValue.goal"
          placeholder="1,000"
          class="w-48"
          @update:model-value="emit('update:modelValue', { ...modelValue, goal: $event as string })"
        >
          <template #prefix>
            <span class="text-sm text-neutral-400">$</span>
          </template>
        </lfx-input>
      </div>

      <!-- Benefits -->
      <div class="flex flex-col gap-3">
        <label class="text-xs font-medium text-neutral-900">Benefits</label>

        <div
          v-for="(benefit, index) in modelValue.benefits"
          :key="index"
          class="flex items-center gap-3"
        >
          <lfx-input
            :model-value="benefit"
            placeholder="E.g. Advisory board seat"
            class="flex-1"
            @update:model-value="updateBenefit(index, $event as string)"
          />
          <lfx-icon-button
            icon="trash-can"
            type="transparent"
            size="small"
            aria-label="Remove benefit"
            @click="removeBenefit(index)"
          />
        </div>

        <lfx-button
          type="transparent"
          size="small"
          icon="plus"
          label="Add benefit"
          class="w-fit"
          @click="addBenefit"
        />
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { SPONSORSHIP_TIER_LABEL, SPONSORSHIP_TIER_ICON_GRADIENT } from '../../../config/donation-options.config';
import type { SponsorshipTierConfig } from '~/types/fundraise.types';
import LfxCheckbox from '~/components/uikit/checkbox/checkbox.vue';
import LfxInput from '~/components/uikit/input/input.vue';
import LfxButton from '~/components/uikit/button/button.vue';
import LfxIconButton from '~/components/uikit/icon-button/icon-button.vue';

const props = defineProps<{
  modelValue: SponsorshipTierConfig;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: SponsorshipTierConfig): void;
}>();

const addBenefit = () => {
  emit('update:modelValue', { ...props.modelValue, benefits: [...props.modelValue.benefits, ''] });
};

const removeBenefit = (index: number) => {
  emit('update:modelValue', {
    ...props.modelValue,
    benefits: props.modelValue.benefits.filter((_, i) => i !== index),
  });
};

const updateBenefit = (index: number, value: string) => {
  emit('update:modelValue', {
    ...props.modelValue,
    benefits: props.modelValue.benefits.map((b, i) => (i === index ? value : b)),
  });
};
</script>

<script lang="ts">
export default {
  name: 'FundraiseDonationTierCard',
};
</script>

<style scoped>
.tier-icon--diamond {
  clip-path: polygon(50% 0, 100% 50%, 50% 100%, 0 50%);
}
</style>
