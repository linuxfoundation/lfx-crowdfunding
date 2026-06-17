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

      <div class="max-w-[720px] w-full mx-auto px-8 py-8">
        <!-- Success state -->
        <fundraise-success
          v-if="submitted"
          :initiative-type="selectedType"
          @done="close()"
        />

        <!-- Multi-step flow -->
        <template v-else>
          <fundraise-header />

          <!-- Body -->
          <div class="flex-1 overflow-y-auto w-full py-8">
            <fundraise-step-type
              v-if="step === 0"
              v-model="selectedType"
            />
            <fundraise-step-details
              v-else-if="step === 1"
              ref="detailsStepRef"
              :initiative-type="selectedType"
            />
          </div>

          <fundraise-footer
            :step="step"
            :continue-label="continueLabel"
            :is-current-step-valid="isCurrentStepValid"
            :submitting="submitting"
            @cancel="close()"
            @previous="previousStep()"
            @continue="handleContinue()"
          />
        </template>
      </div>
    </div>
  </lfx-modal>
</template>

<script setup lang="ts">
import { computed, ref, nextTick, onMounted, onUnmounted } from 'vue';
import FundraiseStepType from './main/fundraise-step-type.vue';
import FundraiseStepDetails from './fundraise-step-details.vue';
import FundraiseHeader from './main/fundraise-header.vue';
import FundraiseFooter from './main/fundraise-footer.vue';
import FundraiseSuccess from './main/fundraise-success.vue';
import { useFundraiseSubmit } from '~/composables/useFundraiseSubmit';
import type { InitiativeType } from '~/types/fundraise.types';
import LfxModal from '~/components/uikit/modal/modal.vue';
import LfxIconButton from '~/components/uikit/icon-button/icon-button.vue';
import { GITHUB_FUNDRAISE_SESSION_KEY, type GitHubFundraiseSession } from '~/composables/useGithubAuth';

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
const submitted = ref(false);

const selectedType = ref<InitiativeType | null>(null);

const detailsStepRef = ref<InstanceType<typeof FundraiseStepDetails> | null>(null);
const { submitting, submitFundraise } = useFundraiseSubmit();

const totalSteps = computed(() => 2);
const isLastStep = computed(() => step.value === totalSteps.value - 1);

const isCurrentStepValid = computed(() => {
  if (step.value === 0) return selectedType.value !== null;
  return detailsStepRef.value?.isCurrentSubStepValid ?? false;
});

const continueLabel = computed(() => {
  if (isLastStep.value && (detailsStepRef.value?.isLastSubStep ?? true)) return 'Submit initiative';
  return 'Continue';
});

// Track whether close was triggered by the browser back button to avoid
// calling history.back() a second time in onUnmounted.
let closedViaPopstate = false;

const handlePopState = () => {
  closedViaPopstate = true;
  close();
};

const close = () => {
  isOpen.value = false;
  step.value = 0;
  submitted.value = false;
  selectedType.value = null;
  detailsStepRef.value?.reset();
};

onMounted(async () => {
  // Push a sentinel history entry so the browser back button triggers popstate
  // instead of navigating away from the page while the drawer is open.
  window.history.pushState({ fundraiseDrawer: true }, document.title);
  window.addEventListener('popstate', handlePopState);

  const raw = sessionStorage.getItem(GITHUB_FUNDRAISE_SESSION_KEY);
  if (!raw) return;
  sessionStorage.removeItem(GITHUB_FUNDRAISE_SESSION_KEY);
  try {
    const session = JSON.parse(raw) as GitHubFundraiseSession;
    selectedType.value = session.initiativeType as InitiativeType;
    step.value = session.step;
    await nextTick();
    if (detailsStepRef.value) {
      detailsStepRef.value.subStep = session.subStep;
      if (session.hostingType === 'github') {
        detailsStepRef.value.projectForm.hostingType = session.hostingType;
      }
    }
  } catch {
    // Corrupted session — start fresh
  }
});

onUnmounted(() => {
  window.removeEventListener('popstate', handlePopState);
  if (!closedViaPopstate) {
    // Drawer was closed by a button — remove the sentinel history entry we pushed.
    window.history.back();
  }
  closedViaPopstate = false;
});

const previousStep = () => {
  if (step.value === 1 && (detailsStepRef.value?.subStep ?? 0) > 0) {
    detailsStepRef.value?.back();
    return;
  }
  if (step.value > 0) step.value--;
};

const handleContinue = async () => {
  if (!isCurrentStepValid.value) return;

  if (step.value === 1 && !(detailsStepRef.value?.isLastSubStep ?? true)) {
    detailsStepRef.value?.advance();
    return;
  }

  if (!isLastStep.value) {
    step.value++;
    return;
  }

  try {
    await submitFundraise(selectedType.value!, {
      projectForm: detailsStepRef.value?.projectForm,
      securityAuditForm: detailsStepRef.value?.securityAuditForm,
      generalFundForm: detailsStepRef.value?.generalFundForm,
      eventForm: detailsStepRef.value?.eventForm,
    });
    submitted.value = true;
    emit('submitted');
  } catch {
    // Error display is handled by useFundraiseSubmit
  }
};
</script>

<script lang="ts">
export default {
  name: 'FundraiseDrawer',
};
</script>
