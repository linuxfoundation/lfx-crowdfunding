<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="flex flex-col gap-6">
    <lfx-field
      label="Funding goal"
      :required="true"
    >
      <lfx-input
        v-model="dollarValue"
        type="number"
        placeholder="0"
        :invalid="$v.goalAmountCents.$error"
        @blur="$v.goalAmountCents.$touch()"
      >
        <template #prefix>
          <span class="text-neutral-500 text-sm">$</span>
        </template>
      </lfx-input>
      <lfx-field-messages :validation="$v.goalAmountCents" />
    </lfx-field>

    <lfx-field label="Funding deadline">
      <lfx-input
        v-model="form.deadline"
        type="date"
      />
    </lfx-field>

    <!-- Event-specific -->
    <template v-if="initiativeType === 'event'">
      <lfx-field label="Expected attendees">
        <lfx-input
          v-model="form.expectedAttendees"
          type="number"
          placeholder="e.g. 250"
        />
      </lfx-field>
    </template>

    <!-- Project-specific -->
    <template v-if="initiativeType === 'project'">
      <div class="rounded-xl border border-neutral-200 p-4 flex flex-col gap-1 bg-neutral-50">
        <p class="text-sm font-medium text-neutral-900">Sponsor tiers</p>
        <p class="text-xs text-neutral-600">
          Default sponsorship tiers (Bronze, Silver, Gold, Platinum) will be generated based on your funding goal. You
          can customise them after your initiative is approved.
        </p>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import useVuelidate from '@vuelidate/core';
import { required, minValue, helpers } from '@vuelidate/validators';
import type { InitiativeType } from './fundraise-step-type.vue';
import LfxField from '~/components/uikit/field/field.vue';
import LfxFieldMessages from '~/components/uikit/field/field-messages.vue';
import LfxInput from '~/components/uikit/input/input.vue';

export interface FundraiseGoalsForm {
  goalAmountCents: number;
  deadline: string;
  expectedAttendees: string;
}

const props = defineProps<{
  modelValue: FundraiseGoalsForm;
  initiativeType: InitiativeType | null;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: FundraiseGoalsForm): void;
}>();

const form = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val),
});

const dollarValue = computed({
  get: () => (form.value.goalAmountCents > 0 ? String(form.value.goalAmountCents / 100) : ''),
  set: (val: string) => {
    const cents = Math.round(parseFloat(val || '0') * 100);
    emit('update:modelValue', { ...form.value, goalAmountCents: isNaN(cents) ? 0 : cents });
  },
});

const rules = {
  goalAmountCents: {
    required: helpers.withMessage('Funding goal is required', required),
    minValue: helpers.withMessage('Goal must be greater than $0', minValue(1)),
  },
};

const $v = useVuelidate(rules, form);

defineExpose({ $v });
</script>

<script lang="ts">
export default {
  name: 'FundraiseStepGoals',
};
</script>
