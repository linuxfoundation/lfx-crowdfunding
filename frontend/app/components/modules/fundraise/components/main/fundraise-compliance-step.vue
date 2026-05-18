<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="flex flex-col gap-5">
    <!-- Compliance Confirmation -->
    <div class="border border-neutral-200 rounded-xl p-6">
      <div class="flex flex-col gap-5">
        <p class="text-sm font-semibold text-neutral-900">
          Compliance Confirmation <span class="text-negative-500">*</span>
        </p>
        <lfx-checkbox
          class="!flex !flex-nowrap !items-start !gap-5 w-full"
          :model-value="modelValue.ofacConfirmed"
          @update:model-value="emit('update:modelValue', { ...modelValue, ofacConfirmed: $event })"
        >
          <span class="text-sm text-neutral-900 leading-5">
            I hereby certify that I am not, and/or the organization I am representing is not: located in Cuba, Iran,
            North Korea, Syria, the Crimea Region of Ukraine, or the Russian-controlled areas of the Donetsk or Luhansk
            regions of Ukraine; owned or controlled by, acting for or on behalf of, or an individual or entity that has
            in the past acted for or on behalf of the Government of Cuba, Iran, North Korea, Syria, or Venezuela; or
            listed as a blocked person by the U.S. Department of the Treasury's Office of Foreign Assets Control (OFAC)
            or directly or indirectly owned 50 percent or more by such a listed person.
          </span>
        </lfx-checkbox>
      </div>
    </div>

    <!-- Terms and Conditions -->
    <div class="border border-neutral-200 rounded-xl p-6">
      <div class="flex flex-col gap-5">
        <div class="flex flex-col gap-2">
          <p class="text-sm font-semibold text-neutral-900">
            Terms and Conditions <span class="text-negative-500">*</span>
          </p>
          <p class="text-xs text-neutral-600 leading-4">
            Before you submit the application to LFX, please review and accept the terms and conditions below.
          </p>
        </div>
        <lfx-checkbox
          class="!flex !flex-nowrap !items-start !gap-5 w-full"
          :model-value="modelValue.termsAccepted"
          @update:model-value="emit('update:modelValue', { ...modelValue, termsAccepted: $event })"
        >
          <span class="text-sm text-neutral-900 leading-5">
            I agree to the
            <a
              href="https://lfx.linuxfoundation.org/platform-use-agreement"
              target="_blank"
              rel="noopener noreferrer"
              class="text-accent-500 underline hover:text-accent-600"
              >LFX Platform Use Agreement</a
            >
            and all terms incorporated therein, including the Service-Specific Use Terms, the Acceptable Use Policy and
            the Privacy Policy.
          </span>
        </lfx-checkbox>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { ComplianceData } from '~/types/fundraise.types';
import LfxCheckbox from '~/components/uikit/checkbox/checkbox.vue';

defineProps<{
  modelValue: ComplianceData;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: ComplianceData): void;
}>();
</script>

<script lang="ts">
export default {
  name: 'FundraiseComplianceStep',
};
</script>
