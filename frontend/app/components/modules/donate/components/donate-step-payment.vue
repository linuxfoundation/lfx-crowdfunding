<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="flex flex-col gap-6">
    <div>
      <p class="text-sm font-medium text-neutral-700 mb-3">Payment method</p>
      <div class="rounded-xl border border-neutral-200 p-5 flex flex-col gap-4">
        <!-- Card number -->
        <lfx-field
          label="Card number"
          :required="true"
        >
          <lfx-input
            :model-value="form.cardNumber"
            placeholder="1234 5678 9123 4567"
            :invalid="$v.cardNumber.$error"
            @update:model-value="onCardNumberInput"
            @keydown="onCardNumberKeydown"
            @paste="onCardNumberPaste"
            @blur="$v.cardNumber.$touch()"
          />
          <lfx-field-messages :validation="$v.cardNumber" />
        </lfx-field>

        <!-- Expiry + CVC -->
        <div class="grid grid-cols-3 gap-4">
          <lfx-field
            class="col-span-2"
            label="Expiry"
            :required="true"
          >
            <lfx-input
              :model-value="form.expiry"
              placeholder="MM/YY"
              :invalid="$v.expiry.$error"
              @update:model-value="onExpiryInput"
              @keydown="onExpiryKeydown"
              @paste="onExpiryPaste"
              @blur="$v.expiry.$touch()"
            />
            <lfx-field-messages :validation="$v.expiry" />
          </lfx-field>

          <lfx-field
            label="CVC"
            :required="true"
          >
            <lfx-input
              :model-value="form.cvc"
              placeholder="123"
              :invalid="$v.cvc.$error"
              @update:model-value="onCvcInput"
              @keydown="onCvcKeydown"
              @paste="onCvcPaste"
              @blur="$v.cvc.$touch()"
            />
            <lfx-field-messages :validation="$v.cvc" />
          </lfx-field>
        </div>

        <!-- Order summary -->
        <div class="border-t border-neutral-200 pt-4 flex flex-col gap-2.5">
          <div class="flex items-center justify-between">
            <span class="text-sm text-neutral-500">Donation to {{ initiativeName }}</span>
            <span class="text-sm text-neutral-900">{{ formattedAmount }}</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-sm text-neutral-500">Fees (LF underwrites)</span>
            <span class="text-sm text-positive-600">$0.00</span>
          </div>
          <div class="flex items-center justify-between border-t border-neutral-200 pt-2.5 mt-0.5">
            <span class="text-sm font-semibold text-neutral-900">Total</span>
            <span class="text-sm font-bold text-neutral-900">{{ formattedAmount }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Tier note -->
    <div
      v-if="tierName"
      class="flex items-center gap-2 text-sm text-neutral-600"
    >
      <span class="size-2 rounded-full bg-warning-500 shrink-0" />
      <span>{{ tierName }} tier benefits included</span>
    </div>

    <!-- Legal -->
    <p class="text-xs text-neutral-400 leading-relaxed">
      All donations are processed by the Linux Foundation, a 501(c)(6) nonprofit. 100% of your contribution goes
      directly to the initiative. LF underwrites all platform and payment fees until $10M.
    </p>
  </div>
</template>

<script setup lang="ts">
import { reactive, watch, computed } from 'vue';
import { useVuelidate } from '@vuelidate/core';
import { required } from '@vuelidate/validators';
import type { DonatePaymentForm } from '#shared/types/donate.types';
import LfxField from '~/components/uikit/field/field.vue';
import LfxFieldMessages from '~/components/uikit/field/field-messages.vue';
import LfxInput from '~/components/uikit/input/input.vue';

const props = defineProps<{
  modelValue: DonatePaymentForm;
  amountCents: number;
  tierName: string | null;
  initiativeName: string;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: DonatePaymentForm): void;
}>();

const form = reactive<DonatePaymentForm>({ ...props.modelValue });

watch(
  () => props.modelValue,
  (val) => Object.assign(form, val),
  { deep: true },
);

watch(form, (val) => emit('update:modelValue', { ...val }), { deep: true });

const rules = {
  cardNumber: { required },
  expiry: { required },
  cvc: { required },
};

const $v = useVuelidate(rules, form);

const formattedAmount = computed(() => {
  const dollars = props.amountCents / 100;
  return dollars >= 1_000 ? `$${(dollars / 1_000).toLocaleString()}K` : `$${dollars.toLocaleString()}`;
});

// Keys that are always allowed regardless of field type
const PASSTHROUGH_KEYS = new Set([
  'Backspace',
  'Delete',
  'Tab',
  'Enter',
  'ArrowLeft',
  'ArrowRight',
  'ArrowUp',
  'ArrowDown',
  'Home',
  'End',
]);

const isNumericKey = (e: KeyboardEvent) => PASSTHROUGH_KEYS.has(e.key) || e.ctrlKey || e.metaKey || /^\d$/.test(e.key);

// --- Card number ---

const formatCardNumber = (digits: string) => digits.replace(/(.{4})(?=.)/g, '$1 ');

const onCardNumberKeydown = (e: KeyboardEvent) => {
  if (!isNumericKey(e)) {
    e.preventDefault();
    return;
  }
  if (/^\d$/.test(e.key) && form.cardNumber.replace(/\D/g, '').length >= 16) {
    e.preventDefault();
  }
};

const onCardNumberInput = (val: string | number) => {
  const digits = String(val).replace(/\D/g, '').slice(0, 16);
  form.cardNumber = formatCardNumber(digits);
};

const onCardNumberPaste = (e: ClipboardEvent) => {
  e.preventDefault();
  const digits = (e.clipboardData?.getData('text') ?? '').replace(/\D/g, '').slice(0, 16);
  form.cardNumber = formatCardNumber(digits);
};

// --- Expiry ---

const applyExpiryFormat = (digits: string, isDeleting: boolean): string => {
  if (digits.length >= 3) return `${digits.slice(0, 2)}/${digits.slice(2)}`;
  if (digits.length === 2 && !isDeleting) return `${digits}/`;
  return digits;
};

const onExpiryKeydown = (e: KeyboardEvent) => {
  if (!isNumericKey(e)) {
    e.preventDefault();
    return;
  }
  if (/^\d$/.test(e.key) && form.expiry.replace(/\D/g, '').length >= 4) {
    e.preventDefault();
  }
};

const onExpiryInput = (val: string | number) => {
  const raw = String(val);
  const isDeleting = raw.length < form.expiry.length;
  const digits = raw.replace(/\D/g, '').slice(0, 4);
  form.expiry = applyExpiryFormat(digits, isDeleting);
};

const onExpiryPaste = (e: ClipboardEvent) => {
  e.preventDefault();
  const digits = (e.clipboardData?.getData('text') ?? '').replace(/\D/g, '').slice(0, 4);
  form.expiry = applyExpiryFormat(digits, false);
};

// --- CVC ---

const onCvcKeydown = (e: KeyboardEvent) => {
  if (!isNumericKey(e)) {
    e.preventDefault();
    return;
  }
  if (/^\d$/.test(e.key) && form.cvc.length >= 4) {
    e.preventDefault();
  }
};

const onCvcInput = (val: string | number) => {
  form.cvc = String(val).replace(/\D/g, '').slice(0, 4);
};

const onCvcPaste = (e: ClipboardEvent) => {
  e.preventDefault();
  form.cvc = (e.clipboardData?.getData('text') ?? '').replace(/\D/g, '').slice(0, 4);
};

defineExpose({ $v });
</script>

<script lang="ts">
export default {
  name: 'DonateStepPayment',
};
</script>
