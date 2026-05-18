<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="flex flex-col gap-4">
    <div class="grid grid-cols-2 gap-6">
      <div class="flex flex-col gap-2">
        <label class="text-xs font-medium text-neutral-900">First name</label>
        <lfx-input
          :model-value="modelValue.firstName"
          placeholder=""
          @update:model-value="update('firstName', $event as string)"
        />
      </div>
      <div class="flex flex-col gap-2">
        <label class="text-xs font-medium text-neutral-900">Last name</label>
        <lfx-input
          :model-value="modelValue.lastName"
          placeholder=""
          @update:model-value="update('lastName', $event as string)"
        />
      </div>
    </div>

    <div class="grid grid-cols-2 gap-6">
      <div class="flex flex-col gap-2">
        <label class="text-xs font-medium text-neutral-900">Email</label>
        <lfx-input
          :model-value="modelValue.email"
          placeholder=""
          @update:model-value="update('email', $event as string)"
        />
      </div>
      <div class="flex flex-col gap-2">
        <label class="text-xs font-medium text-neutral-900">Phone number</label>
        <lfx-input
          :model-value="modelValue.phone"
          placeholder=""
          @update:model-value="update('phone', $event as string)"
        />
      </div>
    </div>

    <div class="flex flex-col gap-2">
      <label class="text-xs font-medium text-neutral-900">Preferred Method of Contact</label>
      <div class="flex items-center gap-3">
        <button
          type="button"
          class="flex items-center gap-1 h-9 px-2.5 py-1 rounded-full border text-sm transition-colors"
          :class="
            modelValue.preferredContact === 'email'
              ? 'bg-accent-100 border-accent-300 text-accent-500'
              : 'bg-white border-neutral-200 text-neutral-900 hover:bg-neutral-50'
          "
          @click="update('preferredContact', 'email')"
        >
          <lfx-icon
            name="envelope"
            type="light"
            :size="12"
          />
          Email
        </button>
        <button
          type="button"
          class="flex items-center gap-1 h-9 px-2.5 py-1 rounded-full border text-sm transition-colors"
          :class="
            modelValue.preferredContact === 'phone'
              ? 'bg-accent-100 border-accent-300 text-accent-500'
              : 'bg-white border-neutral-200 text-neutral-900 hover:bg-neutral-50'
          "
          @click="update('preferredContact', 'phone')"
        >
          <lfx-icon
            name="phone"
            type="light"
            :size="12"
          />
          Phone number
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { ContactPerson } from '~/types/fundraise.types';
import LfxInput from '~/components/uikit/input/input.vue';
import LfxIcon from '~/components/uikit/icon/icon.vue';

const props = defineProps<{
  modelValue: ContactPerson;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: ContactPerson): void;
}>();

const update = <K extends keyof ContactPerson>(key: K, value: ContactPerson[K]) => {
  emit('update:modelValue', { ...props.modelValue, [key]: value });
};
</script>

<script lang="ts">
export default {
  name: 'FundraiseContactPersonGroup',
};
</script>
