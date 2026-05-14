<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <lfx-drawer
    v-model="isOpen"
    position="bottom"
    hide-close-button
  >
    <div class="flex flex-col max-h-[85vh] w-[70vw] mx-auto">
      <!-- Header -->
      <div class="flex items-center justify-between border-b border-neutral-200 px-8 py-6">
        <div class="flex items-center gap-4">
          <div class="relative size-12 shrink-0 overflow-hidden rounded-full bg-neutral-100">
            <img
              v-if="initiative.logoUrl"
              :src="initiative.logoUrl"
              :alt="initiative.name"
              class="size-full object-cover"
            />
            <div
              v-else
              class="flex size-full items-center justify-center"
            >
              <lfx-icon
                name="hands-holding-dollar"
                type="light"
                :size="22"
                class="text-neutral-400"
              />
            </div>
          </div>

          <div class="flex flex-col gap-0.5">
            <div class="flex items-center gap-1.5 text-accent-500">
              <lfx-icon
                name="hands-holding-dollar"
                type="light"
                :size="13"
              />
              <span class="text-xs font-medium">Donate to</span>
            </div>
            <h2 class="text-2xl font-semibold text-neutral-900">
              {{ initiative.name }}
            </h2>
          </div>
        </div>

        <lfx-icon-button
          type="transparent"
          icon="xmark"
          @click="close()"
        />
      </div>

      <!-- Body -->
      <div class="flex-1 overflow-y-auto px-8 py-6">
        <donate-step-amount
          v-if="step === 0"
          v-model="form"
        />
      </div>

      <!-- Footer -->
      <div class="flex items-center justify-between border-t border-neutral-200 px-8 py-4">
        <lfx-button
          type="ghost"
          label="Cancel"
          @click="close()"
        />

        <div class="flex items-center gap-4">
          <span
            v-if="amountSummary"
            class="text-sm text-neutral-600"
          >
            {{ amountSummary }}
          </span>
          <lfx-button
            type="primary"
            label="Continue"
            icon="chevron-right"
            icon-position="right"
            :disabled="!hasSelection"
            :loading="submitting"
            @click="handleContinue()"
          />
        </div>
      </div>
    </div>
  </lfx-drawer>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import DonateStepAmount from './donate-step-amount.vue';
import type { DonateAmountForm } from '#shared/types/donate.types';
import LfxDrawer from '~/components/uikit/drawer/drawer.vue';
import LfxIcon from '~/components/uikit/icon/icon.vue';
import LfxIconButton from '~/components/uikit/icon-button/icon-button.vue';
import LfxButton from '~/components/uikit/button/button.vue';

const props = defineProps<{
  modelValue: boolean;
  initiative: {
    id: string;
    name: string;
    logoUrl?: string;
  };
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void;
  (e: 'submitted'): void;
}>();

const isOpen = computed({
  get: () => props.modelValue,
  set: (val: boolean) => emit('update:modelValue', val),
});

const step = ref(0);
const submitting = ref(false);

const form = ref<DonateAmountForm>({
  tierId: null,
  tierName: null,
  customAmountCents: null,
  amountCents: 0,
});

const hasSelection = computed(() => form.value.amountCents > 0);

const amountSummary = computed(() => {
  if (!hasSelection.value) return null;
  const dollars = form.value.amountCents / 100;
  const formatted = dollars >= 1_000 ? `$${(dollars / 1_000).toLocaleString()}K` : `$${dollars.toLocaleString()}`;
  return form.value.tierName ? `Amount: ${formatted} (${form.value.tierName})` : `Amount: ${formatted}`;
});

const close = () => {
  isOpen.value = false;
  step.value = 0;
  form.value = { tierId: null, tierName: null, customAmountCents: null, amountCents: 0 };
};

const handleContinue = async () => {
  if (!hasSelection.value) return;

  submitting.value = true;
  try {
    await $fetch('/api/donate', {
      method: 'POST',
      body: {
        initiativeId: props.initiative.id,
        tierId: form.value.tierId,
        tierName: form.value.tierName,
        amountCents: form.value.amountCents,
      },
    });
    emit('submitted');
    close();
  } finally {
    submitting.value = false;
  }
};
</script>

<script lang="ts">
export default {
  name: 'DonateDrawer',
};
</script>
