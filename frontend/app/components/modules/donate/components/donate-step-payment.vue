<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="flex flex-col gap-6">
    <div>
      <p class="text-sm font-medium text-neutral-700 mb-3">Payment method</p>
      <div class="rounded-xl border border-neutral-200 p-5 flex flex-col gap-4">
        <!-- Saved card summary — shown when a card is on file and user hasn't chosen to change -->
        <div
          v-if="card && !useDifferentCard"
          class="flex items-center justify-between"
        >
          <div class="flex items-center gap-2">
            <lfx-icon
              name="credit-card"
              type="light"
              :size="16"
              class="text-neutral-500"
            />
            <span class="text-sm text-neutral-900 capitalize"> {{ card.brand }} ···· {{ card.last_four }} </span>
          </div>
          <lfx-button
            type="tertiary"
            size="small"
            label="Use a different card"
            @click="switchToNewCard()"
          />
        </div>

        <!-- Card entry form — shown for new users or when changing card -->
        <template v-if="!card || useDifferentCard">
          <!-- Card number -->
          <lfx-field
            label="Card number"
            :required="true"
          >
            <div
              class="flex items-center border transition-all rounded-md shadow-xs"
              :class="
                cardNumberFocused
                  ? 'border-neutral-900'
                  : cardNumberError
                    ? 'border-negative-600'
                    : 'border-neutral-200'
              "
            >
              <div
                ref="cardNumberContainer"
                class="w-full py-2 px-2"
              />
            </div>
            <lfx-field-message v-if="cardNumberError">{{ cardNumberError }}</lfx-field-message>
          </lfx-field>

          <!-- Expiry + CVC -->
          <div class="grid grid-cols-3 gap-4">
            <lfx-field
              class="col-span-2"
              label="Expiry"
              :required="true"
            >
              <div
                class="flex items-center border transition-all rounded-md shadow-xs"
                :class="
                  cardExpiryFocused
                    ? 'border-neutral-900'
                    : cardExpiryError
                      ? 'border-negative-600'
                      : 'border-neutral-200'
                "
              >
                <div
                  ref="cardExpiryContainer"
                  class="w-full py-2 px-2"
                />
              </div>
              <lfx-field-message v-if="cardExpiryError">{{ cardExpiryError }}</lfx-field-message>
            </lfx-field>

            <lfx-field
              label="CVC"
              :required="true"
            >
              <div
                class="flex items-center border transition-all rounded-md shadow-xs"
                :class="
                  cardCvcFocused ? 'border-neutral-900' : cardCvcError ? 'border-negative-600' : 'border-neutral-200'
                "
              >
                <div
                  ref="cardCvcContainer"
                  class="w-full py-2 px-2"
                />
              </div>
              <lfx-field-message v-if="cardCvcError">{{ cardCvcError }}</lfx-field-message>
            </lfx-field>
          </div>
        </template>

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

    <!-- Stripe / save-card error -->
    <lfx-field-message v-if="stripeError">{{ stripeError }}</lfx-field-message>

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
import { ref, computed, watch, onMounted, onBeforeUnmount, nextTick } from 'vue';
import type { StripeCardNumberElement, StripeCardExpiryElement, StripeCardCvcElement } from '@stripe/stripe-js';
import LfxField from '~/components/uikit/field/field.vue';
import LfxFieldMessage from '~/components/uikit/field/field-message.vue';
import LfxButton from '~/components/uikit/button/button.vue';
import LfxIcon from '~/components/uikit/icon/icon.vue';

const props = defineProps<{
  amountCents: number;
  tierName: string | null;
  initiativeName: string;
}>();

const emit = defineEmits<{
  (e: 'update:complete', value: boolean): void;
}>();

const { getStripe } = useStripe();
const { card, fetchCard, error: saveCardError } = usePaymentAccount();

const useDifferentCard = ref(false);

// DOM refs for Stripe element mounts
const cardNumberContainer = ref<HTMLElement | null>(null);
const cardExpiryContainer = ref<HTMLElement | null>(null);
const cardCvcContainer = ref<HTMLElement | null>(null);

// Stripe element instances
let cardNumberEl: StripeCardNumberElement | null = null;
let cardExpiryEl: StripeCardExpiryElement | null = null;
let cardCvcEl: StripeCardCvcElement | null = null;

// Per-field state
const cardNumberFocused = ref(false);
const cardExpiryFocused = ref(false);
const cardCvcFocused = ref(false);

const cardNumberError = ref('');
const cardExpiryError = ref('');
const cardCvcError = ref('');

const cardNumberComplete = ref(false);
const cardExpiryComplete = ref(false);
const cardCvcComplete = ref(false);

const stripeError = ref('');

const allComplete = computed(() => cardNumberComplete.value && cardExpiryComplete.value && cardCvcComplete.value);
const showCardForm = computed(() => !card.value || useDifferentCard.value);

const formattedAmount = computed(() => {
  const dollars = props.amountCents / 100;
  return dollars >= 1_000 ? `$${(dollars / 1_000).toLocaleString()}K` : `$${dollars.toLocaleString()}`;
});

// Stripe element style — matches LfxInput: text-sm text-neutral-900, placeholder text-neutral-400
const STRIPE_STYLE = {
  base: {
    color: '#0F172A',
    fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif',
    fontSize: '14px',
    lineHeight: '20px',
    '::placeholder': { color: '#94A3B8' },
  },
  invalid: {
    color: '#0F172A',
  },
};

const mountElements = async () => {
  const stripe = await getStripe();
  if (!stripe || !cardNumberContainer.value || !cardExpiryContainer.value || !cardCvcContainer.value) return;

  const elements = stripe.elements();

  cardNumberEl = elements.create('cardNumber', { style: STRIPE_STYLE, placeholder: '1234 5678 9012 3456' });
  cardExpiryEl = elements.create('cardExpiry', { style: STRIPE_STYLE });
  cardCvcEl = elements.create('cardCvc', { style: STRIPE_STYLE });

  cardNumberEl.mount(cardNumberContainer.value);
  cardExpiryEl.mount(cardExpiryContainer.value);
  cardCvcEl.mount(cardCvcContainer.value);

  cardNumberEl.on('change', (e) => {
    cardNumberComplete.value = e.complete;
    cardNumberError.value = e.error?.message ?? '';
    emit('update:complete', allComplete.value);
  });
  cardNumberEl.on('focus', () => {
    cardNumberFocused.value = true;
  });
  cardNumberEl.on('blur', () => {
    cardNumberFocused.value = false;
  });

  cardExpiryEl.on('change', (e) => {
    cardExpiryComplete.value = e.complete;
    cardExpiryError.value = e.error?.message ?? '';
    emit('update:complete', allComplete.value);
  });
  cardExpiryEl.on('focus', () => {
    cardExpiryFocused.value = true;
  });
  cardExpiryEl.on('blur', () => {
    cardExpiryFocused.value = false;
  });

  cardCvcEl.on('change', (e) => {
    cardCvcComplete.value = e.complete;
    cardCvcError.value = e.error?.message ?? '';
    emit('update:complete', allComplete.value);
  });
  cardCvcEl.on('focus', () => {
    cardCvcFocused.value = true;
  });
  cardCvcEl.on('blur', () => {
    cardCvcFocused.value = false;
  });
};

const destroyElements = () => {
  cardNumberEl?.destroy();
  cardExpiryEl?.destroy();
  cardCvcEl?.destroy();
  cardNumberEl = null;
  cardExpiryEl = null;
  cardCvcEl = null;
};

const switchToNewCard = () => {
  useDifferentCard.value = true;
  emit('update:complete', false);
};

// Signal complete to the parent: true when the saved card summary is shown, false otherwise
// (lets the Donate button stay disabled until the new-card form is filled in).
watch(
  [card, useDifferentCard],
  () => {
    emit('update:complete', !showCardForm.value || allComplete.value);
  },
  { immediate: true },
);

// Mount Stripe elements when the card entry form becomes visible; destroy them when hidden
// to avoid dangling mounts after card is saved and the form is removed by v-if.
watch(showCardForm, async (visible) => {
  if (visible && !cardNumberEl) {
    await nextTick(); // wait for v-if containers to render
    await mountElements();
  } else if (!visible) {
    destroyElements();
  }
});

// Surface save-card errors in the Stripe error slot
watch(saveCardError, (msg) => {
  if (msg) stripeError.value = msg;
});

onMounted(async () => {
  await fetchCard();
  if (showCardForm.value) {
    await mountElements();
  }
});

onBeforeUnmount(() => {
  destroyElements();
});

defineExpose({
  getCardNumberEl: (): StripeCardNumberElement | null => cardNumberEl,
  isUsingDifferentCard: (): boolean => useDifferentCard.value,
});
</script>

<script lang="ts">
export default {
  name: 'DonateStepPayment',
};
</script>
