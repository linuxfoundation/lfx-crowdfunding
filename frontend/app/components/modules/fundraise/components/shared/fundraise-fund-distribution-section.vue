<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="border border-neutral-200 rounded-xl p-6">
    <div class="flex flex-col gap-5">
      <div class="flex flex-col gap-2">
        <h2 class="text-base font-semibold text-neutral-900">{{ title }}</h2>
        <p class="text-xs text-neutral-900 leading-4">{{ description }}</p>
      </div>

      <!-- Goal input -->
      <div class="flex flex-col gap-3">
        <label class="text-xs font-medium text-neutral-900">
          {{ goalLabel }} <span class="text-negative-500">*</span>
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

      <!-- Distribution header -->
      <div class="flex flex-col gap-1">
        <p class="text-xs font-medium text-neutral-900">{{ distributionLabel }}</p>
        <p class="text-xs text-neutral-600 leading-4">
          Allocate your funds across categories. Enabled categories must have a percentage greater than zero.
        </p>
      </div>

      <!-- Distribution progress bar: only shown when at least one category is enabled -->
      <div
        v-if="hasEnabledCategories"
        class="flex flex-col gap-1"
      >
        <div class="flex items-center gap-2">
          <div class="flex-1 h-1 rounded-full bg-neutral-200 overflow-hidden">
            <div
              class="h-full rounded-full transition-all"
              :class="totalAllocated > 100 ? 'bg-negative-500' : 'bg-warning-500'"
              :style="{ width: `${Math.min(totalAllocated, 100)}%` }"
            />
          </div>
          <span class="text-xs font-semibold text-neutral-900 shrink-0">{{ totalAllocated }}%</span>
        </div>
        <p
          class="text-xs leading-4"
          :class="totalAllocated > 100 ? 'text-negative-600' : 'text-warning-600'"
        >
          {{ remaining >= 0 ? `${remaining}% remaining` : `${Math.abs(remaining)}% over budget` }}
        </p>
      </div>

      <!-- Categories -->
      <div class="flex flex-col">
        <div
          v-for="(item, index) in modelValue.distribution"
          :key="item.category"
          class="flex items-center gap-5 py-4"
          :class="index < modelValue.distribution.length - 1 ? 'border-b border-neutral-200' : ''"
        >
          <lfx-toggle
            :model-value="item.enabled"
            @update:model-value="toggleCategory(index, $event)"
          />
          <div class="flex-1 flex flex-col gap-1 min-w-0">
            <p class="text-xs text-neutral-900 leading-4">{{ item.label }}</p>
            <p class="text-xs text-neutral-500 leading-4">{{ item.description }}</p>
          </div>
          <div
            v-show="item.enabled"
            class="flex items-center gap-0 shrink-0"
          >
            <div class="w-16">
              <lfx-input
                :model-value="item.percentage > 0 ? String(item.percentage) : ''"
                placeholder="% 0"
                @update:model-value="updatePercentage(index, $event as string)"
              />
            </div>
            <div class="w-16 flex items-center justify-end px-3 text-sm text-neutral-900">
              {{ computedAmount(item.percentage) }}
            </div>
          </div>
          <div
            v-show="!item.enabled"
            class="w-32 shrink-0"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { FundDistributionData, FundDistributionItem } from '~/types/fundraise.types';
import LfxInput from '~/components/uikit/input/input.vue';
import LfxToggle from '~/components/uikit/toggle/toggle.vue';

const props = defineProps<{
  title: string;
  description: string;
  goalLabel: string;
  distributionLabel?: string;
  modelValue: FundDistributionData;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: FundDistributionData): void;
}>();

const distributionLabel = computed(() => props.distributionLabel ?? 'Fund Distribution');

const hasEnabledCategories = computed(() => props.modelValue.distribution.some((item) => item.enabled));

const totalAllocated = computed(() =>
  props.modelValue.distribution.filter((item) => item.enabled).reduce((sum, item) => sum + item.percentage, 0),
);

const remaining = computed(() => 100 - totalAllocated.value);

const goalAmount = computed(() => {
  const n = parseFloat(props.modelValue.goal.replace(/[^0-9.]/g, ''));
  return isNaN(n) ? 0 : n;
});

const computedAmount = (percentage: number): string => {
  const amount = (percentage / 100) * goalAmount.value;
  if (amount === 0) return '$0';
  if (amount >= 1000) return `$${Math.round(amount / 1000)}K`;
  return `$${Math.round(amount)}`;
};

const updateDistribution = (index: number, patch: Partial<FundDistributionItem>) => {
  const distribution = props.modelValue.distribution.map((item, i) => (i === index ? { ...item, ...patch } : item));
  emit('update:modelValue', { ...props.modelValue, distribution });
};

const toggleCategory = (index: number, enabled: boolean) => {
  updateDistribution(index, { enabled, percentage: 0 });
};

const updatePercentage = (index: number, raw: string) => {
  const pct = parseInt(raw.replace(/[^0-9]/g, ''), 10);
  updateDistribution(index, { percentage: isNaN(pct) ? 0 : Math.min(pct, 100) });
};
</script>

<script lang="ts">
export default {
  name: 'FundraiseFundDistributionSection',
};
</script>
