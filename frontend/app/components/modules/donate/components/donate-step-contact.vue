<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="flex flex-col gap-3">
    <donate-add-org-modal
      v-model="showAddOrgModal"
      :organization="editingOrg ?? undefined"
      @created="onOrgCreated"
      @updated="showAddOrgModal = false"
    />

    <donate-delete-org-modal
      v-model="showDeleteModal"
      :organization="deletingOrg"
      @deleted="onOrgDeleted"
    />

    <p class="text-xs font-medium text-neutral-900">I'm donating as</p>

    <div class="border border-neutral-200 rounded-xl w-full">
      <!-- Individual row -->
      <div
        class="flex gap-3 items-center p-6 w-full cursor-pointer"
        @click="selectedKey = 'individual'"
      >
        <lfx-avatar
          :src="user?.picture ?? undefined"
          type="member"
          size="normal"
        />
        <div class="flex-1 min-w-0">
          <p class="text-sm font-semibold text-neutral-900 truncate">{{ user?.name ?? 'You' }}</p>
          <p class="text-xs text-neutral-600">Individual</p>
        </div>
        <lfx-radio
          v-model="selectedKey"
          value="individual"
          name="donor-type"
        />
      </div>

      <!-- Skeleton rows while loading -->
      <template v-if="loading">
        <div class="h-px bg-neutral-200" />
        <div class="flex gap-3 items-center p-6 w-full">
          <lfx-skeleton
            width="40px"
            height="40px"
          />
          <div class="flex flex-col gap-1.5 flex-1">
            <lfx-skeleton
              width="140px"
              height="14px"
            />
            <lfx-skeleton
              width="90px"
              height="12px"
            />
          </div>
        </div>
      </template>

      <!-- Organization rows -->
      <template
        v-for="org in organizations"
        :key="org.id"
      >
        <div class="h-px bg-neutral-200" />
        <div
          class="flex gap-3 items-center p-6 w-full cursor-pointer"
          @click="selectedKey = org.id"
        >
          <lfx-avatar
            :src="org.avatarUrl || undefined"
            type="organization"
            size="normal"
          />
          <div class="flex-1 min-w-0">
            <p class="text-sm font-semibold text-neutral-900 truncate">{{ org.name }}</p>
            <p class="text-xs text-neutral-600">Organization</p>
          </div>
          <div class="flex gap-4 items-center shrink-0">
            <div class="flex gap-4">
              <lfx-icon-button
                icon="pen"
                type="outline"
                size="medium"
                aria-label="Edit organization"
                @click.stop="openEditModal(org)"
              />
              <lfx-icon-button
                icon="trash-can"
                type="outline"
                size="medium"
                aria-label="Delete organization"
                @click.stop="openDeleteModal(org)"
              />
            </div>
            <lfx-radio
              v-model="selectedKey"
              :value="org.id"
              name="donor-type"
            />
          </div>
        </div>
      </template>
    </div>

    <lfx-link-button
      icon="plus"
      label="Add organization"
      variant="accent"
      class="self-start mb-4"
      @click="
        editingOrg = null;
        showAddOrgModal = true;
      "
    />
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, watch, computed, onMounted } from 'vue';
import DonateAddOrgModal from './donate-add-org-modal.vue';
import DonateDeleteOrgModal from './donate-delete-org-modal.vue';
import LfxAvatar from '~/components/uikit/avatar/avatar.vue';
import LfxLinkButton from '~/components/uikit/link-button/link-button.vue';
import LfxRadio from '~/components/uikit/radio/radio.vue';
import LfxSkeleton from '~/components/uikit/skeleton/skeleton.vue';
import type { DonateContactForm } from '#shared/types/donate.types';
import type { Organization } from '#shared/types/organization.types';

const props = defineProps<{
  modelValue: DonateContactForm;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: DonateContactForm): void;
}>();

const { user } = useAuth();
const { organizations, loading, fetchOrganizations } = useOrganizations();

const form = reactive<DonateContactForm>({ ...props.modelValue });
const showAddOrgModal = ref(false);
const editingOrg = ref<Organization | null>(null);
const showDeleteModal = ref(false);
const deletingOrg = ref<Organization | null>(null);

const openEditModal = (org: Organization) => {
  editingOrg.value = org;
  showAddOrgModal.value = true;
};

const openDeleteModal = (org: Organization) => {
  deletingOrg.value = org;
  showDeleteModal.value = true;
};

const onOrgDeleted = (id: string) => {
  if (form.organizationId === id) {
    form.donorType = 'individual';
    form.organizationId = null;
  }
};

watch(
  () => props.modelValue,
  (val) => Object.assign(form, val),
  { deep: true },
);

watch(form, (val) => emit('update:modelValue', { ...val }), { deep: true });

const selectedKey = computed({
  get: () => (form.donorType === 'individual' ? 'individual' : (form.organizationId ?? 'individual')),
  set: (val: string | number | boolean) => {
    if (val === 'individual') {
      form.donorType = 'individual';
      form.organizationId = null;
    } else {
      form.donorType = 'organization';
      form.organizationId = val as string;
    }
  },
});

const onOrgCreated = (org: Organization) => {
  form.donorType = 'organization';
  form.organizationId = org.id;
};

onMounted(() => {
  fetchOrganizations();
});
</script>

<script lang="ts">
export default {
  name: 'DonateStepContact',
};
</script>
