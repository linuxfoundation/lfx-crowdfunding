<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <lfx-card class="p-6 flex flex-col gap-5">
    <!-- Entry -->
    <template v-if="step === 'form'">
      <lfx-field
        label="Type"
        required
      >
        <div
          class="flex flex-col gap-3"
          role="radiogroup"
          aria-label="Type"
        >
          <lfx-radio
            v-for="option in kindOptions"
            :key="option.value"
            v-model="form.kind"
            :value="option.value"
            name="offline-entry-kind"
          >
            <span class="flex flex-col">
              <span class="text-sm text-neutral-900">{{ option.label }}</span>
              <span class="text-xs text-neutral-500">{{ option.description }}</span>
            </span>
          </lfx-radio>
        </div>
      </lfx-field>

      <lfx-field
        :label="kindOption.orgLabel"
        :required="kindOption.orgRequired"
      >
        <div
          v-if="form.organization"
          class="flex items-center gap-3 mb-2"
        >
          <lfx-avatar
            type="organization"
            size="small"
          />
          <p class="flex-1 min-w-0 text-sm text-neutral-900 truncate">
            {{ form.organization.name }}
            <span class="text-xs text-neutral-500">{{ form.organization.domain }}</span>
          </p>
          <lfx-link-button
            v-if="!kindOption.orgRequired"
            label="Remove"
            variant="accent"
            @click="form.organization = null"
          />
        </div>
        <org-picker v-model="form.organization" />
        <lfx-field-message
          v-if="form.kind === 'payout'"
          type="info"
        >
          Optional: the donor whose money this pays out. The Financials tab shows the payee, not the organization.
        </lfx-field-message>
      </lfx-field>

      <lfx-field
        v-if="form.kind === 'payout'"
        label="Paid to"
        required
      >
        <lfx-input
          v-model="form.paidTo"
          aria-label="Paid to"
          placeholder="e.g. Jane Doe"
          :maxlength="PAID_TO_MAX_LENGTH"
        />
        <lfx-field-message type="info">Shown publicly as the expense's name on the Financials tab.</lfx-field-message>
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
          <lfx-field-message
            v-else-if="form.kind !== 'donation'"
            type="info"
          >
            Enter it as a positive amount; it's taken out of the balance.
          </lfx-field-message>
        </lfx-field>

        <lfx-field
          :label="kindOption.dateLabel"
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
        v-if="form.kind !== 'donation'"
        label="Reason"
        required
      >
        <lfx-textarea
          v-model="form.reason"
          aria-label="Reason"
          placeholder="e.g. Mentee withdrew, CNCF Fall 2026"
          rows="2"
          :maxlength="REASON_MAX_LENGTH"
        />
        <lfx-field-message type="info">Shown publicly next to the entry on the Financials tab.</lfx-field-message>
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
          The reference finance uses for this entry, e.g. the invoice or spreadsheet row. It must be unique: the Ledger
          keeps one entry per reference.
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
        <dt class="text-neutral-500">Type</dt>
        <dd class="text-neutral-900">{{ kindOption.label }}</dd>
        <dt class="text-neutral-500">Initiative</dt>
        <dd class="text-neutral-900">{{ initiative.name }}</dd>
        <template v-if="form.organization">
          <dt class="text-neutral-500">{{ kindOption.orgLabel }}</dt>
          <dd class="text-neutral-900">{{ form.organization.name }} ({{ form.organization.domain }})</dd>
        </template>
        <template v-if="form.kind === 'payout'">
          <dt class="text-neutral-500">Paid to</dt>
          <dd class="text-neutral-900">{{ form.paidTo.trim() }}</dd>
        </template>
        <dt class="text-neutral-500">Amount</dt>
        <dd
          class="font-semibold"
          :class="form.kind === 'donation' ? 'text-neutral-900' : 'text-negative-600'"
        >
          {{ form.kind === 'donation' ? '' : '-' }}{{ formatTypedDollars(form.amount) }}
        </dd>
        <dt class="text-neutral-500">{{ kindOption.dateLabel }}</dt>
        <dd class="text-neutral-900">{{ form.receivedOn ? toIsoDate(form.receivedOn) : '' }}</dd>
        <dt class="text-neutral-500">Category</dt>
        <dd class="text-neutral-900">{{ form.category }}</dd>
        <template v-if="form.kind !== 'donation'">
          <dt class="text-neutral-500">Reason</dt>
          <dd class="text-neutral-900 break-words">{{ form.reason.trim() }}</dd>
        </template>
        <dt class="text-neutral-500">Source reference</dt>
        <dd class="text-neutral-900 break-all">{{ form.sourceTxnId.trim() }}</dd>
      </dl>
      <lfx-field-message :type="form.kind === 'donation' ? 'info' : 'warning'">
        {{ kindOption.reviewNotice }}
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
          :label="kindOption.submitLabel"
          type="primary"
          :loading="submitting"
          @click="submit"
        />
      </div>
    </template>

    <!-- Recorded -->
    <template v-else-if="outcome">
      <div class="flex flex-col gap-3 items-center text-center py-4">
        <lfx-icon
          :name="outcome.icon"
          type="regular"
          :size="40"
          :class="outcome.iconClass"
        />
        <p class="text-base font-semibold text-neutral-900">{{ outcome.title }}</p>
        <p
          class="text-sm"
          :class="outcome.messageClass"
        >
          {{ outcome.message }}
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
  OFFLINE_ENTRY_KIND_OPTIONS,
  PAID_TO_MAX_LENGTH,
  REASON_MAX_LENGTH,
  adjustmentErrorMessage,
  adjustmentOutcome,
  entryKindOption,
  isOfflineEntryFormComplete,
  toLedgerAdjustmentRequest,
} from '../config/ledger-adjustment.config';
import {
  emptyOfflineDonationForm,
  formatTypedDollars,
  isValidDollarAmount,
  offlineCategoryOptions,
  offlineDonationErrorMessage,
  recordedOutcome,
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
import LfxLinkButton from '~/components/uikit/link-button/link-button.vue';
import LfxRadio from '~/components/uikit/radio/radio.vue';
import LfxSelect from '~/components/uikit/select/select.vue';
import LfxTextarea from '~/components/uikit/textarea/textarea.vue';
import type { OfflineDonationForm, OfflineDonationStep, RecordedOutcome } from '~/types/offline-donation.types';
import type { InitiativeDetail } from '#shared/types/initiative-detail.types';
import type { LedgerAdjustmentResult } from '#shared/types/ledger-adjustment.types';
import type { OfflineDonationResult } from '#shared/types/offline-donation.types';

const props = defineProps<{ initiative: InitiativeDetail }>();
const emit = defineEmits<{ (e: 'recorded'): void }>();

const kindOptions = OFFLINE_ENTRY_KIND_OPTIONS;

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
const outcome = ref<RecordedOutcome | null>(null);
const today = new Date();

const kindOption = computed(() => entryKindOption(form.kind));
const amountInvalid = computed(() => form.amount.trim() !== '' && !isValidDollarAmount(form.amount));
const complete = computed(() => isOfflineEntryFormComplete(form));

async function submit() {
  submitting.value = true;
  submitError.value = null;
  try {
    if (form.kind === 'donation') {
      const recorded = await $fetch<OfflineDonationResult>('/api/offline-donations', {
        method: 'POST',
        body: toOfflineDonationRequest(props.initiative.id, form),
      });
      outcome.value = recordedOutcome(recorded, props.initiative.name);
    } else {
      const recorded = await $fetch<LedgerAdjustmentResult>('/api/ledger-adjustments', {
        method: 'POST',
        body: toLedgerAdjustmentRequest(props.initiative.id, form),
      });
      outcome.value = adjustmentOutcome(recorded, props.initiative.name);
    }
    step.value = 'done';
    emit('recorded');
  } catch (err: unknown) {
    const e = err as { statusCode?: number; status?: number; statusMessage?: string; data?: { message?: string } };
    const message = form.kind === 'donation' ? offlineDonationErrorMessage : adjustmentErrorMessage;
    submitError.value = message(e?.statusCode ?? e?.status, e?.data?.message ?? e?.statusMessage);
  } finally {
    submitting.value = false;
  }
}

function reset() {
  const kind = form.kind;
  Object.assign(form, emptyOfflineDonationForm(categoryOptions.value[0]), { kind });
  outcome.value = null;
  submitError.value = null;
  step.value = 'form';
}
</script>

<script lang="ts">
export default {
  name: 'OfflineDonationForm',
};
</script>
