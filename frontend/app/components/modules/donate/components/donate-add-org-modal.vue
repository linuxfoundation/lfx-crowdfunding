<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <lfx-modal
    v-model="isOpen"
    width="600px"
  >
    <div class="flex flex-col gap-8 p-6 relative">
      <!-- Header -->
      <div class="flex items-start justify-between">
        <h2 class="text-xl font-bold text-neutral-900">
          {{ organization ? 'Edit organization' : 'Add organization' }}
        </h2>
        <lfx-icon-button
          type="outline"
          icon="xmark"
          @click="close()"
        />
      </div>

      <!-- Form fields -->
      <div class="flex flex-col gap-6">
        <!-- Organization name -->
        <lfx-field
          label="Organization name"
          :required="true"
        >
          <lfx-input
            v-model="name"
            placeholder=""
            :invalid="nameTouched && !name.trim()"
            @blur="nameTouched = true"
          />
          <lfx-field-message v-if="nameTouched && !name.trim()"> Organization name is required </lfx-field-message>
        </lfx-field>

        <!-- Organization logo -->
        <div class="flex flex-col gap-3">
          <label class="text-xs font-medium text-neutral-900">
            Organization logo <span class="text-negative-500">*</span>
          </label>

          <!-- Uploading -->
          <div
            v-if="uploading"
            class="border border-dashed border-neutral-300 rounded-xl px-5 py-6 flex items-center gap-5"
          >
            <div
              class="size-12 rounded-full bg-neutral-100 border border-neutral-200 flex items-center justify-center shrink-0 overflow-hidden relative"
            >
              <img
                v-if="previewSrc"
                :src="previewSrc"
                alt="Logo preview"
                class="size-full object-contain opacity-50"
              />
              <lfx-icon
                v-else
                name="image"
                type="light"
                :size="20"
                class="text-neutral-400"
              />
              <div class="absolute inset-0 flex items-center justify-center">
                <lfx-icon
                  name="spinner"
                  type="solid"
                  :size="16"
                  class="text-neutral-500 animate-spin"
                />
              </div>
            </div>
            <span class="text-sm text-neutral-500">Uploading…</span>
          </div>

          <!-- Has logo -->
          <div
            v-else-if="logoUrl"
            class="border border-dashed border-neutral-300 rounded-xl px-5 py-6 flex items-center gap-5"
          >
            <div
              class="size-12 rounded-full bg-neutral-100 border border-neutral-200 flex items-center justify-center shrink-0 overflow-hidden"
            >
              <img
                :src="logoUrl"
                alt="Organization logo"
                class="size-full object-contain"
              />
            </div>
            <div class="flex-1 flex items-center justify-between min-w-0">
              <span class="text-sm text-neutral-900 truncate">{{ localFileName }}</span>
              <div class="flex items-center gap-4 shrink-0">
                <button
                  type="button"
                  class="flex items-center gap-1 text-xs font-semibold text-neutral-900 hover:text-negative-500 transition-colors"
                  @click="removeLogo()"
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

          <!-- Empty / drop zone -->
          <div
            v-else
            class="border border-dashed border-neutral-300 rounded-xl px-5 py-6 flex items-center justify-center gap-5 cursor-pointer hover:bg-neutral-50 transition-colors"
            :class="{ 'bg-neutral-50': isDragging }"
            @click="fileInput?.click()"
            @dragover.prevent="isDragging = true"
            @dragleave.prevent="isDragging = false"
            @drop.prevent="onDrop"
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
              <p class="text-xs text-neutral-400 leading-4">JPG or PNG ・ Max 2MB ・ 600×600px</p>
            </div>
          </div>

          <p
            v-if="uploadError"
            class="text-xs text-negative-500"
          >
            {{ uploadError }}
          </p>

          <input
            ref="fileInput"
            type="file"
            accept="image/png,image/jpeg"
            class="hidden"
            @change="onFileChange"
          />
        </div>
      </div>

      <!-- Footer -->
      <div class="flex items-center justify-end gap-4">
        <lfx-button
          type="outline"
          label="Cancel"
          @click="close()"
        />
        <lfx-button
          type="primary"
          :label="organization ? 'Save changes' : 'Add organization'"
          :disabled="!isFormValid || submitting"
          :loading="submitting"
          @click="submit()"
        />
      </div>
    </div>
  </lfx-modal>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue';
import type { Organization } from '#shared/types/organization.types';
import LfxModal from '~/components/uikit/modal/modal.vue';
import LfxIconButton from '~/components/uikit/icon-button/icon-button.vue';
import LfxButton from '~/components/uikit/button/button.vue';
import LfxField from '~/components/uikit/field/field.vue';
import LfxFieldMessage from '~/components/uikit/field/field-message.vue';
import LfxInput from '~/components/uikit/input/input.vue';
import LfxIcon from '~/components/uikit/icon/icon.vue';

const props = defineProps<{
  modelValue: boolean;
  organization?: Organization;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void;
  (e: 'created', org: Organization): void;
  (e: 'updated', org: Organization): void;
}>();

const isOpen = computed({
  get: () => props.modelValue,
  set: (val: boolean) => emit('update:modelValue', val),
});

const { createOrganization, updateOrganization } = useOrganizations();
const { uploading, error: uploadError, uploadLogo } = useLogoUpload();

const name = ref('');
const nameTouched = ref(false);
const logoUrl = ref('');
const previewSrc = ref<string | null>(null);
const localFileName = ref('');
const isDragging = ref(false);
const submitting = ref(false);
const fileInput = ref<HTMLInputElement | null>(null);

const isFormValid = computed(() => name.value.trim() !== '' && logoUrl.value !== '');

watch(
  () => props.modelValue,
  (open) => {
    if (open && props.organization) {
      name.value = props.organization.name;
      logoUrl.value = props.organization.avatarUrl ?? '';
      previewSrc.value = props.organization.avatarUrl ?? null;
      localFileName.value = '';
      nameTouched.value = false;
    }
  },
);

const close = () => {
  isOpen.value = false;
  name.value = '';
  nameTouched.value = false;
  logoUrl.value = '';
  previewSrc.value = null;
  localFileName.value = '';
  isDragging.value = false;
  if (fileInput.value) fileInput.value.value = '';
};

const handleFile = async (file: File) => {
  previewSrc.value = URL.createObjectURL(file);
  localFileName.value = file.name;
  const url = await uploadLogo(file);
  if (url) {
    logoUrl.value = url;
  } else {
    previewSrc.value = null;
    localFileName.value = '';
  }
};

const onFileChange = (event: Event) => {
  const file = (event.target as HTMLInputElement).files?.[0];
  if (file) handleFile(file);
};

const onDrop = (event: DragEvent) => {
  isDragging.value = false;
  const file = event.dataTransfer?.files?.[0];
  if (file) handleFile(file);
};

const removeLogo = () => {
  logoUrl.value = '';
  previewSrc.value = null;
  localFileName.value = '';
  if (fileInput.value) fileInput.value.value = '';
};

const submit = async () => {
  nameTouched.value = true;
  if (!isFormValid.value) return;
  submitting.value = true;
  try {
    if (props.organization) {
      const org = await updateOrganization(props.organization.id, name.value.trim(), logoUrl.value);
      if (org) {
        emit('updated', org);
        close();
      }
    } else {
      const org = await createOrganization(name.value.trim(), logoUrl.value);
      if (org) {
        emit('created', org);
        close();
      }
    }
  } finally {
    submitting.value = false;
  }
};
</script>

<script lang="ts">
export default {
  name: 'DonateAddOrgModal',
};
</script>
