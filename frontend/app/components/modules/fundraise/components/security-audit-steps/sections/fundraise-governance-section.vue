<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="border border-neutral-200 rounded-xl p-6">
    <div class="flex flex-col gap-5">
      <h2 class="text-base font-semibold text-neutral-900">Governance</h2>

      <!-- Code of Conduct URL -->
      <div class="flex flex-col gap-3">
        <div class="flex flex-col gap-1">
          <div class="flex items-center justify-between">
            <label class="text-xs font-medium text-neutral-900">Code of Conduct URL</label>
            <a
              href="https://www.contributor-covenant.org"
              target="_blank"
              rel="noopener noreferrer"
              class="flex items-center gap-1 text-xs font-medium text-accent-500 hover:text-accent-600"
            >
              Start from a template
              <lfx-icon
                name="arrow-up-right-from-square"
                type="light"
                :size="12"
              />
            </a>
          </div>
          <p class="text-xs text-neutral-600 leading-4">
            If you don't provide one, the Contributor Covenant will be used by default.
          </p>
        </div>
        <lfx-input
          :model-value="modelValue.codeOfConductUrl"
          placeholder="https://example.org/code-of-conduct"
          @update:model-value="emit('update:modelValue', { ...modelValue, codeOfConductUrl: $event as string })"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { SecurityAuditFormData } from '~/types/fundraise.types';
import LfxIcon from '~/components/uikit/icon/icon.vue';
import LfxInput from '~/components/uikit/input/input.vue';

defineProps<{
  modelValue: SecurityAuditFormData;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: SecurityAuditFormData): void;
}>();
</script>

<script lang="ts">
export default {
  name: 'FundraiseGovernanceSection',
};
</script>
