<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="border border-neutral-200 rounded-xl p-6">
    <div class="flex flex-col gap-5">
      <h2 class="text-base font-semibold text-neutral-900">Contact Information</h2>

      <div class="flex flex-col gap-10">
        <!-- Primary Contact -->
        <div class="flex flex-col gap-4">
          <div class="flex items-center gap-3">
            <span class="text-xs font-semibold text-neutral-500 uppercase whitespace-nowrap">Primary Contact</span>
            <div class="flex-1 h-px bg-neutral-200" />
          </div>
          <fundraise-contact-person-group
            :model-value="modelValue.primaryContact"
            @update:model-value="emit('update:modelValue', { ...modelValue, primaryContact: $event })"
          />
        </div>

        <!-- Secondary Contact -->
        <div class="flex flex-col gap-4">
          <div class="flex items-center gap-3">
            <span class="text-xs font-semibold text-neutral-500 uppercase whitespace-nowrap">Secondary Contact</span>
            <div class="flex-1 h-px bg-neutral-200" />
          </div>
          <fundraise-contact-person-group
            :model-value="modelValue.secondaryContact"
            @update:model-value="emit('update:modelValue', { ...modelValue, secondaryContact: $event })"
          />
        </div>

        <!-- Technical Lead -->
        <div class="flex flex-col gap-4">
          <div class="flex flex-col gap-1">
            <div class="flex items-center gap-3">
              <span class="text-xs font-semibold text-neutral-500 uppercase whitespace-nowrap">Technical Lead</span>
              <div class="flex-1 h-px bg-neutral-200" />
            </div>
            <p class="text-xs text-neutral-600 leading-4">(If different from primary or secondary contact)</p>
          </div>
          <fundraise-contact-person-group
            :model-value="modelValue.technicalLead"
            @update:model-value="emit('update:modelValue', { ...modelValue, technicalLead: $event })"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import FundraiseContactPersonGroup from './fundraise-contact-person-group.vue';
import type { SecurityAuditFormData } from '~/types/fundraise.types';

defineProps<{
  modelValue: SecurityAuditFormData;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: SecurityAuditFormData): void;
}>();
</script>

<script lang="ts">
export default {
  name: 'FundraiseContactSection',
};
</script>
