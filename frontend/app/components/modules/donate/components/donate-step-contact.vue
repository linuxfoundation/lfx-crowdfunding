<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="flex flex-col gap-6">
    <!-- Donor type toggle -->
    <div>
      <p class="text-sm font-medium text-neutral-700 mb-3">I'm donating as</p>
      <div class="flex gap-2">
        <lfx-button
          :type="form.donorType === 'individual' ? 'primary' : 'outline'"
          button-style="pill"
          icon="user"
          label="Individual"
          @click="setDonorType('individual')"
        />
        <lfx-button
          :type="form.donorType === 'company' ? 'primary' : 'outline'"
          button-style="pill"
          icon="building"
          label="Company"
          @click="setDonorType('company')"
        />
      </div>
    </div>

    <!-- Individual fields -->
    <template v-if="form.donorType === 'individual'">
      <lfx-field
        label="Full name"
        :required="true"
      >
        <lfx-tooltip
          class="!w-full"
          placement="top-start"
          :content="AUTH_FIELD_TOOLTIP"
          :disabled="!user?.name"
        >
          <lfx-input
            v-model="form.fullName"
            placeholder=""
            :disabled="!!user?.name"
            :invalid="$v.fullName.$error"
            @blur="$v.fullName.$touch()"
          />
        </lfx-tooltip>
        <lfx-field-messages :validation="$v.fullName" />
      </lfx-field>

      <lfx-field
        label="Email"
        :required="true"
      >
        <lfx-tooltip
          class="!w-full"
          placement="top-start"
          :content="AUTH_FIELD_TOOLTIP"
          :disabled="!user?.email"
        >
          <lfx-input
            v-model="form.email"
            type="email"
            placeholder=""
            :disabled="!!user?.email"
            :invalid="$v.email.$error"
            @blur="$v.email.$touch()"
          />
        </lfx-tooltip>
        <lfx-field-messages :validation="$v.email" />
      </lfx-field>
    </template>

    <!-- Company fields -->
    <template v-else>
      <div class="grid md:grid-cols-2 grid-cols-1 gap-4">
        <lfx-field
          label="Company name"
          :required="true"
        >
          <lfx-input
            v-model="form.companyName"
            placeholder=""
            :invalid="$v.companyName.$error"
            @blur="$v.companyName.$touch()"
          />
          <lfx-field-messages :validation="$v.companyName" />
        </lfx-field>

        <lfx-field
          label="Contact name"
          :required="true"
        >
          <lfx-tooltip
            class="!w-full"
            placement="top-start"
            :content="AUTH_FIELD_TOOLTIP"
            :disabled="!user?.name"
          >
            <lfx-input
              v-model="form.contactName"
              placeholder=""
              :disabled="!!user?.name"
              :invalid="$v.contactName.$error"
              @blur="$v.contactName.$touch()"
            />
          </lfx-tooltip>
          <lfx-field-messages :validation="$v.contactName" />
        </lfx-field>
      </div>

      <lfx-field
        label="Email"
        :required="true"
      >
        <lfx-tooltip
          class="!w-full"
          placement="top-start"
          :content="AUTH_FIELD_TOOLTIP"
          :disabled="!user?.email"
        >
          <lfx-input
            v-model="form.email"
            type="email"
            placeholder=""
            :disabled="!!user?.email"
            :invalid="$v.email.$error"
            @blur="$v.email.$touch()"
          />
        </lfx-tooltip>
        <lfx-field-messages :validation="$v.email" />
      </lfx-field>

      <!-- Hiding this for now until we can implement this -->
      <!-- <lfx-checkbox v-model="form.needsInvoice"> I need an invoice for this donation </lfx-checkbox> -->

      <lfx-field
        v-if="form.needsInvoice"
        label="PO Number"
      >
        <lfx-input
          v-model="form.poNumber"
          placeholder=""
        />
      </lfx-field>
    </template>
  </div>
</template>

<script setup lang="ts">
import { reactive, watch, computed } from 'vue';
import { useVuelidate } from '@vuelidate/core';
import { required, email as emailValidator } from '@vuelidate/validators';
import type { DonateContactForm, DonorType } from '#shared/types/donate.types';
import LfxButton from '~/components/uikit/button/button.vue';
import LfxField from '~/components/uikit/field/field.vue';
import LfxFieldMessages from '~/components/uikit/field/field-messages.vue';
import LfxInput from '~/components/uikit/input/input.vue';
import LfxTooltip from '~/components/uikit/tooltip/tooltip.vue';

const AUTH_FIELD_TOOLTIP = 'Need to change something? Go to your LF ID';

const props = defineProps<{
  modelValue: DonateContactForm;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: DonateContactForm): void;
}>();

const { user } = useAuth();

const form = reactive<DonateContactForm>({ ...props.modelValue });

watch(
  () => props.modelValue,
  (val) => Object.assign(form, val),
  { deep: true },
);

watch(
  user,
  (authUser) => {
    if (authUser?.name) {
      form.fullName = authUser.name;
      form.contactName = authUser.name;
    }
    if (authUser?.email) {
      form.email = authUser.email;
    }
  },
  { immediate: true },
);

watch(form, (val) => emit('update:modelValue', { ...val }), { deep: true });

const rules = computed(() => ({
  email: { required, email: emailValidator },
  ...(form.donorType === 'individual'
    ? { fullName: { required } }
    : { companyName: { required }, contactName: { required } }),
}));

const $v = useVuelidate(rules, form);

const setDonorType = (type: DonorType) => {
  form.donorType = type;
  $v.value.$reset();
};

defineExpose({ $v });
</script>

<script lang="ts">
export default {
  name: 'DonateStepContact',
};
</script>
