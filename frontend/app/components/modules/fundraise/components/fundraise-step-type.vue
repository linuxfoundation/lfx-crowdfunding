<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="flex flex-col gap-5">
    <div class="flex flex-col gap-2">
      <h2 class="text-base font-semibold text-neutral-900">Choose initiative type</h2>
      <p class="text-xs text-neutral-900">This determines the setup flow and how your initiative is categorized.</p>
    </div>

    <div class="border border-neutral-200 rounded-xl overflow-hidden">
      <div
        v-for="(option, index) in INITIATIVE_TYPES"
        :key="option.value"
      >
        <button
          type="button"
          class="w-full flex items-start gap-3 p-6 text-left transition-colors"
          :class="modelValue === option.value ? 'bg-accent-50' : 'bg-white hover:bg-neutral-50'"
          @click="$emit('update:modelValue', option.value)"
        >
          <!-- Icon -->
          <div
            class="shrink-0 size-10 rounded-full bg-white border border-neutral-200 shadow-sm flex items-center justify-center"
          >
            <lfx-icon
              :name="option.icon"
              type="light"
              :size="20"
              class="text-neutral-900"
            />
          </div>

          <!-- Label + description -->
          <div class="flex-1 min-w-0 flex flex-col gap-2 justify-center">
            <p class="text-sm font-semibold text-neutral-900 leading-5">
              {{ option.label }}
            </p>
            <p class="text-xs text-neutral-600 leading-4">
              {{ option.description }}
            </p>
          </div>

          <!-- Radio indicator -->
          <div
            class="shrink-0 size-4 rounded-full border flex items-center justify-center mt-0.5"
            :class="modelValue === option.value ? 'bg-accent-500 border-accent-500' : 'bg-white border-neutral-300'"
          >
            <div
              v-if="modelValue === option.value"
              class="size-1.5 rounded-full bg-white"
            />
          </div>
        </button>

        <div
          v-if="index < INITIATIVE_TYPES.length - 1"
          class="border-b border-neutral-200"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import LfxIcon from '~/components/uikit/icon/icon.vue';

export type InitiativeType = 'project' | 'security_audit' | 'general_fund' | 'event';

interface InitiativeTypeOption {
  value: InitiativeType;
  label: string;
  description: string;
  icon: string;
}

const INITIATIVE_TYPES: InitiativeTypeOption[] = [
  {
    value: 'project',
    label: 'Project',
    description: 'Dedicated funding for a specific open source project.',
    icon: 'code',
  },
  {
    value: 'security_audit',
    label: 'OSTIF Security Audit',
    description: 'Independent, third-party security audit through OSTIF.',
    icon: 'box-magnifying-glass',
  },
  {
    value: 'general_fund',
    label: 'General Fund',
    description: 'Flexible funding for infrastructure, travel, mentorship, and operations.',
    icon: 'piggy-bank',
  },
  {
    value: 'event',
    label: 'Event / Meetup',
    description: 'Fund conferences, hackathons, meetups, and speaker programs.',
    icon: 'calendar',
  },
];

defineProps<{
  modelValue: InitiativeType | null;
}>();

defineEmits<{
  (e: 'update:modelValue', value: InitiativeType): void;
}>();
</script>

<script lang="ts">
export default {
  name: 'FundraiseStepType',
};
</script>
