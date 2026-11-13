<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <lfx-card class="p-6 flex flex-col gap-5">
    <!-- Entry -->
    <template v-if="step === 'form'">
      <p class="text-sm font-semibold text-neutral-900">Donation</p>

      <lfx-field
        label="Organization"
        required
      >
        <div
          v-if="form.organization"
          class="flex items-center gap-3 mb-2"
        >
          <lfx-avatar
            type="organization"
            size="small"
          />
          <p class="text-sm text-neutral-900 truncate">
            {{ form.organization.name }}
            <span class="text-xs text-neutral-500">{{ form.organization.domain }}</span>
          </p>
        </div>
        <org-picker v-model="form.organization" />
      </lfx-field>

      <div class="grid sm:grid-cols-2 gap-4">
        <lfx-field
          label="Amount (USD)"
          required
        >
          <lfx-input
            v-model="form.amount"
            aria-label="Amount in US dollars"
            placeholder="1300.00"
            inputmode="decimal"
            :invalid="amountInvalid"
          >
            <template #prefix>$</template>
          </lfx-input>
          <lfx-field-message
            v-if="amountInvalid"
            type="error"
          >
            Enter a dollar amount like 1300 or 1300.50
          </lfx-field-message>
        </lfx-field>

        <lfx-field
          label="Date received"
          required
        >
          <lfx-datepicker
            v-model="form.receivedOn"
            :max-date="today"
            date-format="yy-mm-dd"
            placeholder="YYYY-MM-DD"
            show-icon
            fluid
          />
        </lfx-field>
      </div>

      <lfx-field
        label="Category"
        required
      >
        <lfx-select v-model="form.category">
          <lfx-dropdown-item
            v-for="option in categoryOptions"
            :key="option"
            :value="option"
            :label="option"
          />
        </lfx-select>
      </lfx-field>

      <lfx-field
        label="Source reference"
        required
      >
        <lfx-input
          v-model="form.sourceTxnId"
          aria-label="Source reference"
          placeholder="e.g. LFX Mentorship CNCF Fall 2026 Row 52"
          maxlength="255"
        />
        <lfx-field-message type="info">
          The reference finance uses for this payment, e.g. the invoice or spreadsheet row. It must be unique: the
          Ledger keeps one entry per reference.
        </lfx-field-message>
      </lfx-field>

      <div class="flex justify-end">
        <lfx-button
          label="Review"
          type="primary"
          :disabled="!complete"
          @click="step = 'review'"
        />
      </div>
    </template>

    <!-- Confirmation summary -->
    <template v-else-if="step === 'review'">
      <p class="text-sm font-semibold text-neutral-900">Check the details</p>
      <dl class="grid grid-cols-[max-content_1fr] gap-x-6 gap-y-2 text-sm">
        <dt class="text-neutral-500">Initiative</dt>
        <dd class="text-neutral-900">{{ initiative.name }}</dd>
        <dt class="text-neutral-500">Organization</dt>
        <dd class="text-neutral-900">{{ form.organization?.name }} ({{ form.organization?.domain }})</dd>
        <dt class="text-neutral-500">Amount</dt>
        <dd class="text-neutral-900 font-semibold">{{ formatTypedDollars(form.amount) }}</dd>
        <dt class="text-neutral-500">Date received</dt>
        <dd class="text-neutral-900">{{ form.receivedOn ? toIsoDate(form.receivedOn) : '' }}</dd>
        <dt class="text-neutral-500">Category</dt>
        <dd class="text-neutral-900">{{ form.category }}</dd>
        <dt class="text-neutral-500">Source reference</dt>
        <dd class="text-neutral-900 break-all">{{ form.sourceTxnId.trim() }}</dd>
      </dl>
      <lfx-field-message type="info">
        This posts the donation to the Ledger. It can't be edited or removed from here afterwards.
      </lfx-field-message>
      <lfx-field-message
        v-if="submitError"
        type="error"
      >
        {{ submitError }}
      </lfx-field-message>
      <div class="flex justify-end gap-3">
        <lfx-button
          label="Edit"
          type="tertiary"
          :disabled="submitting"
          @click="step = 'form'"
        />
        <lfx-button
          label="Record donation"
          type="primary"
          :loading="submitting"
          @click="submit"
        />
      </div>
    </template>

    <!-- Recorded -->
    <template v-else-if="recorded">
      <div class="flex flex-col gap-3 items-center text-center py-4">
        <lfx-icon
          name="circle-check"
          type="regular"
          :size="40"
          class="text-green-500"
        />
        <p class="text-base font-semibold text-neutral-900">Donation recorded</p>
        <p class="text-sm text-neutral-600">
          {{ formatCents(recorded.amountCents) }} from {{ recorded.organizationName }} to {{ initiative.name }}. The
          initiative's totals and donor list update within the hour.
        </p>
        <lfx-button
          label="Record another"
          type="secondary"
          @click="reset"
        />
      </div>
    </template>
  </lfx-card>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue';
import {
  emptyOfflineDonationForm,
  formatCents,
  formatTypedDollars,
  isOfflineDonationFormComplete,
  isValidDollarAmount,
  offlineCategoryOptions,
  offlineDonationErrorMessage,
  toIsoDate,
  toOfflineDonationRequest,
} from '../config/offline-donation.config';
import OrgPicker from '~/components/shared/components/org-picker/org-picker.vue';
import LfxAvatar from '~/components/uikit/avatar/avatar.vue';
import LfxButton from '~/components/uikit/button/button.vue';
import LfxCard from '~/components/uikit/card/card.vue';
import LfxDatepicker from '~/components/uikit/datepicker/datepicker.vue';
import LfxDropdownItem from '~/components/uikit/dropdown/dropdown-item.vue';
import LfxField from '~/components/uikit/field/field.vue';
import LfxFieldMessage from '~/components/uikit/field/field-message.vue';
import LfxIcon from '~/components/uikit/icon/icon.vue';
import LfxInput from '~/components/uikit/input/input.vue';
import LfxSelect from '~/components/uikit/select/select.vue';
import type { OfflineDonationForm, OfflineDonationStep } from '~/types/offline-donation.types';
import type { InitiativeDetail } from '#shared/types/initiative-detail.types';
import type { OfflineDonation } from '#shared/types/offline-donation.types';

const props = defineProps<{ initiative: InitiativeDetail }>();
const emit = defineEmits<{ (e: 'recorded', donation: OfflineDonation): void }>();

const categoryOptions = computed(() =>
  offlineCategoryOptions(
    props.initiative.initiativeType,
    (props.initiative.fundingGoals ?? []).map((g) => g.name),
  ),
);

const form = reactive<OfflineDonationForm>(emptyOfflineDonationForm(categoryOptions.value[0]));
const step = ref<OfflineDonationStep>('form');
const submitting = ref(false);
const submitError = ref<string | null>(null);
const recorded = ref<OfflineDonation | null>(null);
const today = new Date();

const amountInvalid = computed(() => form.amount.trim() !== '' && !isValidDollarAmount(form.amount));
const complete = computed(() => isOfflineDonationFormComplete(form));

async function submit() {
  submitting.value = true;
  submitError.value = null;
  try {
    recorded.value = await $fetch<OfflineDonation>('/api/offline-donations', {
      method: 'POST',
      body: toOfflineDonationRequest(props.initiative.id, form),
    });
    step.value = 'done';
    emit('recorded', recorded.value);
  } catch (err: unknown) {
    const e = err as { statusCode?: number; status?: number; statusMessage?: string; data?: { message?: string } };
    submitError.value = offlineDonationErrorMessage(e?.statusCode ?? e?.status, e?.data?.message ?? e?.statusMessage);
  } finally {
    submitting.value = false;
  }
}

function reset() {
  Object.assign(form, emptyOfflineDonationForm(categoryOptions.value[0]));
  recorded.value = null;
  submitError.value = null;
  step.value = 'form';
}
</script>

<script lang="ts">
export default {
  name: 'OfflineDonationForm',
};
</script>
