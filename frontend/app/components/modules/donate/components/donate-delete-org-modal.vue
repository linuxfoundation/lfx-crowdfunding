<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <lfx-modal
    v-model="isOpen"
    width="440px"
  >
    <div class="flex flex-col gap-6 p-6">
      <div class="flex items-start justify-between">
        <h2 class="text-lg font-bold text-neutral-900">Delete organization?</h2>
        <lfx-icon-button
          type="outline"
          icon="xmark"
          @click="isOpen = false"
        />
      </div>
      <p class="text-sm text-neutral-600">
        Are you sure you want to delete
        <span class="font-semibold text-neutral-900">{{ organization?.name }}</span
        >? This action cannot be undone.
      </p>
      <div class="flex justify-end gap-3">
        <lfx-button
          label="Cancel"
          type="secondary"
          @click="isOpen = false"
        />
        <lfx-button
          label="Delete"
          type="primary"
          :loading="deleting"
          @click="handleDelete"
        />
      </div>
    </div>
  </lfx-modal>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import type { Organization } from '#shared/types/organization.types';
import LfxModal from '~/components/uikit/modal/modal.vue';
import LfxIconButton from '~/components/uikit/icon-button/icon-button.vue';
import LfxButton from '~/components/uikit/button/button.vue';

const props = defineProps<{
  modelValue: boolean;
  organization: Organization | null;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void;
  (e: 'deleted', id: string): void;
}>();

const { deleteOrganization } = useOrganizations();
const deleting = ref(false);

const isOpen = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val),
});

const handleDelete = async () => {
  if (!props.organization) return;
  deleting.value = true;
  const ok = await deleteOrganization(props.organization.id);
  deleting.value = false;
  if (ok) {
    emit('deleted', props.organization.id);
    isOpen.value = false;
  }
};
</script>

<script lang="ts">
export default {
  name: 'DonateDeleteOrgModal',
};
</script>
