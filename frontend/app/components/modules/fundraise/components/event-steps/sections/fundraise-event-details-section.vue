<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="border border-neutral-200 rounded-xl p-6">
    <div class="flex flex-col gap-5">
      <h2 class="text-base font-semibold text-neutral-900">Event Details</h2>

      <!-- Event name -->
      <div class="flex flex-col gap-3">
        <div class="flex items-center justify-between">
          <label class="text-xs font-medium text-neutral-900">
            Event name <span class="text-negative-500">*</span>
          </label>
          <span class="text-xs text-neutral-500">{{ modelValue.name.length }}/100</span>
        </div>
        <lfx-input
          :model-value="modelValue.name"
          placeholder="My event"
          @update:model-value="update('name', $event as string)"
        />
      </div>

      <!-- Event summary -->
      <div class="flex flex-col gap-3">
        <div class="flex items-center justify-between">
          <label class="text-xs font-medium text-neutral-900">
            Event summary <span class="text-negative-500">*</span>
          </label>
          <span
            class="text-xs"
            :class="modelValue.elevatorPitch.length >= 5000 ? 'text-negative-500' : 'text-neutral-500'"
          >
            {{ modelValue.elevatorPitch.length }}/5000
          </span>
        </div>
        <lfx-textarea
          :model-value="modelValue.elevatorPitch"
          placeholder="Briefly introduce your event..."
          class="h-[72px]"
          :maxlength="5000"
          @update:model-value="update('elevatorPitch', $event as string)"
        />
      </div>

      <!-- Topic / Category -->
      <div class="flex flex-col gap-3">
        <label class="text-xs font-medium text-neutral-900">
          Topic / Category <span class="text-negative-500">*</span>
        </label>
        <fundraise-project-topic-select
          :model-value="modelValue.topics"
          @update:model-value="update('topics', $event)"
        />
      </div>

      <!-- Website URL -->
      <div class="flex flex-col gap-3">
        <div class="flex flex-col gap-1">
          <label class="text-xs font-medium text-neutral-900">Website URL</label>
          <p class="text-xs text-neutral-600 leading-4">
            The website where interested attendees can learn more about your event.
          </p>
        </div>
        <lfx-input
          :model-value="modelValue.websiteUrl"
          placeholder="https://events.example.org"
          @update:model-value="update('websiteUrl', $event as string)"
        />
      </div>

      <!-- Registration URL -->
      <div class="flex flex-col gap-3">
        <div class="flex flex-col gap-1">
          <label class="text-xs font-medium text-neutral-900">
            Registration URL <span class="text-negative-500">*</span>
          </label>
          <p class="text-xs text-neutral-600 leading-4">
            The specific registration URL where people can sign up for your event.
          </p>
        </div>
        <lfx-input
          :model-value="modelValue.registrationUrl"
          placeholder="https://eventbrite.com/e/your-event"
          @update:model-value="update('registrationUrl', $event as string)"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import FundraiseProjectTopicSelect from '../../project-steps/details/fundraise-project-topic-select.vue';
import type { EventFormData } from '~/types/fundraise.types';
import LfxInput from '~/components/uikit/input/input.vue';
import LfxTextarea from '~/components/uikit/textarea/textarea.vue';

const props = defineProps<{
  modelValue: EventFormData;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: EventFormData): void;
}>();

const update = <K extends keyof EventFormData>(key: K, value: EventFormData[K]) => {
  emit('update:modelValue', { ...props.modelValue, [key]: value });
};
</script>

<script lang="ts">
export default {
  name: 'FundraiseEventDetailsSection',
};
</script>
