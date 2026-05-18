<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="border border-neutral-200 rounded-xl p-6">
    <div class="flex flex-col gap-5">
      <h2 class="text-base font-semibold text-neutral-900">Compliance &amp; Governance</h2>

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
          @update:model-value="emit('update:modelValue', { ...modelValue, ciiProjectId: $event as string })"
        />
      </div>

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
            All projects must have a published Code of Conduct. If you don't provide one, the Contributor Covenant will
            be used.
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
import type { ProjectDetailsData } from '~/types/fundraise.types';
import LfxIcon from '~/components/uikit/icon/icon.vue';
import LfxInput from '~/components/uikit/input/input.vue';

defineProps<{
  modelValue: ProjectDetailsData;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: ProjectDetailsData): void;
}>();
</script>

<script lang="ts">
export default {
  name: 'FundraiseProjectComplianceSection',
};
</script>
