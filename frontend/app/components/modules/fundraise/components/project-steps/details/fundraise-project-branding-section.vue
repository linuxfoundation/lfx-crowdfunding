<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="border border-neutral-200 rounded-xl p-6">
    <div class="flex flex-col gap-5">
      <h2 class="text-base font-semibold text-neutral-900">Branding</h2>

      <div class="flex flex-col gap-3">
        <label class="text-xs font-medium text-neutral-900">
          Project Logo <span class="text-negative-500">*</span>
        </label>

        <!-- Has logo state -->
        <div
          v-if="modelValue.logoFileName"
          class="border border-dashed border-neutral-300 rounded-xl px-5 py-6 flex items-center gap-5"
        >
          <div
            class="size-12 rounded-full bg-neutral-100 border border-neutral-200 flex items-center justify-center shrink-0"
          >
            <lfx-icon
              name="image"
              type="light"
              :size="20"
              class="text-neutral-400"
            />
          </div>
          <div class="flex-1 flex items-center justify-between min-w-0">
            <span class="text-sm text-neutral-900 truncate">{{ modelValue.logoFileName }}</span>
            <div class="flex items-center gap-4 shrink-0">
              <button
                type="button"
                class="flex items-center gap-1 text-xs font-semibold text-neutral-900 hover:text-negative-500 transition-colors"
                @click="removeLogo"
              >
                <lfx-icon
                  name="trash-can"
                  type="light"
                  :size="13"
                />
                Remove
              </button>
              <button
                type="button"
                class="flex items-center gap-1.5 text-xs font-semibold text-neutral-900 border border-neutral-200 rounded-full px-2 py-1 hover:bg-neutral-50 transition-colors"
                @click="fileInput?.click()"
              >
                <lfx-icon
                  name="cloud-upload"
                  type="light"
                  :size="13"
                />
                Choose different
              </button>
            </div>
          </div>
        </div>

        <!-- Empty state -->
        <div
          v-else
          class="border border-dashed border-neutral-300 rounded-xl px-5 py-6 flex items-center justify-center gap-5 cursor-pointer hover:bg-neutral-50 transition-colors"
          @click="fileInput?.click()"
        >
          <lfx-icon
            name="image"
            type="solid"
            :size="44"
            class="text-neutral-200 shrink-0"
          />
          <div class="flex flex-col gap-2">
            <p class="text-sm text-neutral-900 leading-5">
              <span class="text-accent-500">Click to upload</span> or drag and drop
            </p>
            <p class="text-xs text-neutral-400 leading-4">JPG or PNG ・ Max 2MB ・ 600x600px</p>
          </div>
        </div>

        <input
          ref="fileInput"
          type="file"
          accept="image/png,image/jpeg,image/svg+xml"
          class="hidden"
          @change="onFileChange"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import type { ProjectDetailsData } from '~/types/fundraise.types';
import LfxIcon from '~/components/uikit/icon/icon.vue';

const props = defineProps<{
  modelValue: ProjectDetailsData;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: ProjectDetailsData): void;
}>();

const fileInput = ref<HTMLInputElement | null>(null);

const onFileChange = (event: Event) => {
  const file = (event.target as HTMLInputElement).files?.[0];
  if (file) {
    emit('update:modelValue', { ...props.modelValue, logoFileName: file.name });
  }
};

const removeLogo = () => {
  emit('update:modelValue', { ...props.modelValue, logoFileName: '' });
  if (fileInput.value) fileInput.value.value = '';
};
</script>

<script lang="ts">
export default {
  name: 'FundraiseProjectBrandingSection',
};
</script>
