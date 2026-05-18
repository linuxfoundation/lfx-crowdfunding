<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="border border-neutral-200 rounded-xl p-6">
    <div class="flex flex-col gap-5">
      <h2 class="text-base font-semibold text-neutral-900">Security Details</h2>

      <!-- CII Project ID -->
      <div class="flex flex-col gap-3">
        <div class="flex flex-col gap-1">
          <div class="flex items-center justify-between">
            <label class="text-xs font-medium text-neutral-900">CII Project ID</label>
            <a
              href="https://bestpractices.coreinfrastructure.org"
              target="_blank"
              rel="noopener noreferrer"
              class="flex items-center gap-1 text-xs font-medium text-accent-500 hover:text-accent-600"
            >
              Apply for CII Best Practices Badge
              <lfx-icon
                name="arrow-up-right-from-square"
                type="light"
                :size="12"
              />
            </a>
          </div>
          <p class="text-xs text-neutral-600 leading-4">
            Security is our top priority. All projects must participate in the CII Best Practices badge program within
            90 days.
          </p>
        </div>
        <lfx-input
          :model-value="modelValue.ciiProjectId"
          placeholder=""
          @update:model-value="update('ciiProjectId', $event as string)"
        />
      </div>

      <!-- License Type -->
      <div class="flex flex-col gap-3">
        <div class="flex flex-col gap-1">
          <label class="text-xs font-medium text-neutral-900">License Type</label>
          <p class="text-xs text-neutral-600 leading-4">
            Under what software license(s) does your project operate? Enter 'None' if none.
          </p>
        </div>
        <lfx-input
          :model-value="modelValue.licenseType"
          placeholder="e.g. Apache 2.0, MIT, GPL-3.0"
          @update:model-value="update('licenseType', $event as string)"
        />
      </div>

      <!-- Current Security Strategy -->
      <div class="flex flex-col gap-3">
        <label class="text-xs font-medium text-neutral-900">Current Security Strategy</label>
        <lfx-textarea
          :model-value="modelValue.currentSecurityStrategy"
          placeholder="Briefly describe your current approach to software security. Do you use fuzzers? Internal software reviews?"
          class="h-[72px]"
          @update:model-value="update('currentSecurityStrategy', $event as string)"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { SecurityAuditFormData } from '~/types/fundraise.types';
import LfxIcon from '~/components/uikit/icon/icon.vue';
import LfxInput from '~/components/uikit/input/input.vue';
import LfxTextarea from '~/components/uikit/textarea/textarea.vue';

const props = defineProps<{
  modelValue: SecurityAuditFormData;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: SecurityAuditFormData): void;
}>();

const update = <K extends keyof SecurityAuditFormData>(key: K, value: SecurityAuditFormData[K]) => {
  emit('update:modelValue', { ...props.modelValue, [key]: value });
};
</script>

<script lang="ts">
export default {
  name: 'FundraiseSecurityDetailsSection',
};
</script>
