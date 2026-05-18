<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="border border-neutral-200 rounded-xl p-6">
    <div class="flex flex-col gap-5">
      <div class="flex flex-col gap-2">
        <h2 class="text-base font-semibold text-neutral-900">Beneficiaries</h2>
        <p class="text-xs text-neutral-900 leading-4">
          You will be automatically added as a beneficiary. Add others who can submit expenses below.
        </p>
      </div>

      <div class="flex flex-col">
        <div
          v-for="(beneficiary, index) in modelValue.beneficiaries"
          :key="index"
          class="flex items-center gap-3 py-4"
          :class="index < modelValue.beneficiaries.length - 1 ? 'border-b border-neutral-200' : ''"
        >
          <lfx-input
            :model-value="beneficiary.name"
            placeholder="Full name"
            class="flex-1"
            @update:model-value="updateBeneficiary(index, 'name', $event as string)"
          />
          <lfx-input
            :model-value="beneficiary.email"
            placeholder="Email address"
            class="flex-1"
            @update:model-value="updateBeneficiary(index, 'email', $event as string)"
          />
          <button
            type="button"
            class="size-9 shrink-0 flex items-center justify-center rounded-full hover:bg-neutral-100 transition-colors text-neutral-900"
            @click="removeBeneficiary(index)"
          >
            <lfx-icon
              name="trash-can"
              type="light"
              :size="16"
            />
          </button>
        </div>
      </div>

      <button
        type="button"
        class="flex items-center gap-1.5 text-sm font-medium text-accent-500 hover:text-accent-600 transition-colors w-fit"
        @click="addBeneficiary"
      >
        <lfx-icon
          name="plus"
          type="light"
          :size="16"
        />
        Add beneficiary
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Beneficiary, ProjectDetailsData } from '~/types/fundraise.types';
import LfxIcon from '~/components/uikit/icon/icon.vue';
import LfxInput from '~/components/uikit/input/input.vue';

const props = defineProps<{
  modelValue: ProjectDetailsData;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: ProjectDetailsData): void;
}>();

const addBeneficiary = () => {
  const beneficiaries: Beneficiary[] = [...props.modelValue.beneficiaries, { name: '', email: '' }];
  emit('update:modelValue', { ...props.modelValue, beneficiaries });
};

const removeBeneficiary = (index: number) => {
  const beneficiaries = props.modelValue.beneficiaries.filter((_, i) => i !== index);
  emit('update:modelValue', { ...props.modelValue, beneficiaries });
};

const updateBeneficiary = (index: number, field: keyof Beneficiary, value: string) => {
  const beneficiaries = props.modelValue.beneficiaries.map((b, i) => (i === index ? { ...b, [field]: value } : b));
  emit('update:modelValue', { ...props.modelValue, beneficiaries });
};
</script>

<script lang="ts">
export default {
  name: 'FundraiseProjectBeneficiariesSection',
};
</script>
