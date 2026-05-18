<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <lfx-modal
    v-model="isOpen"
    type="cover"
  >
    <div class="flex flex-col h-full">
      <!-- Close button -->
      <lfx-icon-button
        class="absolute top-6 right-6 z-10"
        type="outline"
        icon="xmark"
        @click="close()"
      />

      <!-- Success state -->
      <template v-if="submitted">
        <div class="flex-1 overflow-y-auto flex items-center justify-center">
          <div class="max-w-[720px] w-full mx-auto px-8 py-12 flex flex-col items-center gap-6 text-center">
            <div class="size-16 rounded-full bg-positive-50 flex items-center justify-center">
              <lfx-icon
                name="check"
                type="solid"
                :size="28"
                class="text-positive-600"
              />
            </div>
            <div class="flex flex-col gap-2">
              <h2 class="font-secondary font-light text-2xl text-neutral-900">Initiative submitted!</h2>
              <p class="text-sm text-neutral-600">
                Your
                <strong>{{ selectedTypeLabel }}</strong>
                initiative has been submitted for review. You'll receive an email once it's approved.
              </p>
            </div>
            <lfx-button
              type="primary"
              label="Done"
              @click="close()"
            />
          </div>
        </div>
      </template>

      <!-- Multi-step flow -->
      <template v-else>
        <!-- Header -->
        <div class="border-b border-neutral-200 shrink-0">
          <div class="max-w-[720px] w-full mx-auto px-8 py-6">
            <div class="flex flex-col gap-1">
              <div class="flex items-center gap-2 text-accent-600">
                <lfx-icon
                  name="box-dollar"
                  type="light"
                  :size="15"
                />
                <span class="text-xs font-medium leading-4">Fundraise</span>
              </div>
              <h1 class="font-secondary font-light text-2xl text-black">Start a new initiative</h1>
            </div>
          </div>
        </div>

        <!-- Body -->
        <div class="flex-1 overflow-y-auto">
          <div class="max-w-[720px] w-full mx-auto px-8 py-8">
            <fundraise-step-type
              v-if="step === 0"
              v-model="selectedType"
            />
            <fundraise-step-details
              v-else-if="step === 1"
              ref="detailsStepRef"
              v-model="detailsForm"
              :initiative-type="selectedType"
            />
            <fundraise-step-goals
              v-else-if="step === 2"
              ref="goalsStepRef"
              v-model="goalsForm"
              :initiative-type="selectedType"
            />
          </div>
        </div>

        <!-- Footer -->
        <div class="border-t border-neutral-200 shrink-0">
          <div class="max-w-[720px] w-full mx-auto px-8 py-8 flex items-center justify-between">
            <lfx-button
              type="outline"
              label="Cancel"
              @click="close()"
            />

            <div class="flex items-center gap-5">
              <lfx-icon-button
                v-if="step > 0"
                type="outline"
                icon="chevron-left"
                @click="previousStep()"
              />
              <lfx-button
                type="primary"
                :label="continueLabel"
                icon="angle-right"
                icon-position="right"
                :disabled="!isCurrentStepValid"
                :loading="submitting"
                @click="handleContinue()"
              />
            </div>
          </div>
        </div>
      </template>
    </div>
  </lfx-modal>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import FundraiseStepType, { type InitiativeType } from './fundraise-step-type.vue';
import FundraiseStepDetails, { type FundraiseDetailsForm } from './fundraise-step-details.vue';
import FundraiseStepGoals, { type FundraiseGoalsForm } from './fundraise-step-goals.vue';
import LfxModal from '~/components/uikit/modal/modal.vue';
import LfxIcon from '~/components/uikit/icon/icon.vue';
import LfxIconButton from '~/components/uikit/icon-button/icon-button.vue';
import LfxButton from '~/components/uikit/button/button.vue';

const TYPE_LABELS: Record<InitiativeType, string> = {
  project: 'Project',
  security_audit: 'OSTIF Security Audit',
  general_fund: 'General Fund',
  event: 'Event / Meetup',
};

// Types with a goals step (step 2)
const MULTI_STEP_TYPES: InitiativeType[] = ['project', 'event'];

const props = defineProps<{
  modelValue: boolean;
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
const submitted = ref(false);

const selectedType = ref<InitiativeType | null>(null);

const detailsForm = ref<FundraiseDetailsForm>({
  name: '',
  description: '',
  githubUrl: '',
  tags: '',
  auditScope: '',
  eventDate: '',
  location: '',
  eventbriteUrl: '',
});

const goalsForm = ref<FundraiseGoalsForm>({
  goalAmountCents: 0,
  deadline: '',
  expectedAttendees: '',
});

const detailsStepRef = ref<InstanceType<typeof FundraiseStepDetails> | null>(null);
const goalsStepRef = ref<InstanceType<typeof FundraiseStepGoals> | null>(null);

const hasGoalsStep = computed(() => selectedType.value !== null && MULTI_STEP_TYPES.includes(selectedType.value));

const totalSteps = computed(() => (hasGoalsStep.value ? 3 : 2));
const isLastStep = computed(() => step.value === totalSteps.value - 1);
const selectedTypeLabel = computed(() => (selectedType.value ? TYPE_LABELS[selectedType.value] : ''));

const isCurrentStepValid = computed(() => {
  if (step.value === 0) return selectedType.value !== null;
  if (step.value === 1) {
    const f = detailsForm.value;
    const baseValid = f.name.trim().length > 0 && f.description.trim().length > 0;
    if (selectedType.value === 'security_audit') {
      return baseValid && f.githubUrl.trim().length > 0 && f.auditScope.trim().length > 0;
    }
    return baseValid;
  }
  return goalsForm.value.goalAmountCents > 0;
});

const continueLabel = computed(() => {
  if (isLastStep.value) return 'Submit';
  return 'Continue';
});

const close = () => {
  isOpen.value = false;
  step.value = 0;
  submitted.value = false;
  selectedType.value = null;
  detailsForm.value = {
    name: '',
    description: '',
    githubUrl: '',
    tags: '',
    auditScope: '',
    eventDate: '',
    location: '',
    eventbriteUrl: '',
  };
  goalsForm.value = { goalAmountCents: 0, deadline: '', expectedAttendees: '' };
};

const previousStep = () => {
  if (step.value > 0) step.value--;
};

const handleContinue = async () => {
  if (!isCurrentStepValid.value) return;

  if (!isLastStep.value) {
    if (step.value === 1 && detailsStepRef.value?.$v) {
      detailsStepRef.value.$v.$touch();
      if (detailsStepRef.value.$v.$invalid) return;
    }
    step.value++;
    return;
  }

  if (step.value === 2 && goalsStepRef.value?.$v) {
    goalsStepRef.value.$v.$touch();
    if (goalsStepRef.value.$v.$invalid) return;
  }

  submitting.value = true;
  try {
    await $fetch('/api/fundraise', {
      method: 'POST',
      body: {
        initiativeType: selectedType.value,
        details: detailsForm.value,
        goals: hasGoalsStep.value ? goalsForm.value : null,
      },
    });
    submitted.value = true;
    emit('submitted');
  } catch {
    // API errors surface via $fetch
  } finally {
    submitting.value = false;
  }
};
</script>

<script lang="ts">
export default {
  name: 'FundraiseDrawer',
};
</script>
