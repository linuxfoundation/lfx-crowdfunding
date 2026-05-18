<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="flex flex-col gap-6">
    <lfx-field
      label="Initiative name"
      :required="true"
    >
      <lfx-input
        v-model="form.name"
        placeholder="e.g. Kubernetes Security Audit"
        :invalid="$v.name.$error"
        @blur="$v.name.$touch()"
      />
      <lfx-field-messages :validation="$v.name" />
    </lfx-field>

    <lfx-field
      label="Description"
      :required="true"
    >
      <lfx-textarea
        v-model="form.description"
        placeholder="Briefly describe the purpose and goals of this initiative"
        :invalid="$v.description.$error"
        @blur="$v.description.$touch()"
      />
      <lfx-field-messages :validation="$v.description" />
    </lfx-field>

    <!-- Project-specific -->
    <template v-if="initiativeType === 'project'">
      <lfx-field label="GitHub URL">
        <lfx-input
          v-model="form.githubUrl"
          placeholder="https://github.com/org/repo"
          :invalid="$v.githubUrl.$error"
          @blur="$v.githubUrl.$touch()"
        />
        <lfx-field-messages :validation="$v.githubUrl" />
      </lfx-field>

      <lfx-field label="Tags">
        <lfx-input
          v-model="form.tags"
          placeholder="e.g. security, containers, kubernetes"
        />
        <p class="text-xs text-neutral-500 mt-1">Comma-separated keywords to help people find your initiative</p>
      </lfx-field>
    </template>

    <!-- Security audit-specific -->
    <template v-else-if="initiativeType === 'security_audit'">
      <lfx-field
        label="Project GitHub URL"
        :required="true"
      >
        <lfx-input
          v-model="form.githubUrl"
          placeholder="https://github.com/org/repo"
          :invalid="$v.githubUrl.$error"
          @blur="$v.githubUrl.$touch()"
        />
        <lfx-field-messages :validation="$v.githubUrl" />
      </lfx-field>

      <lfx-field
        label="Audit scope"
        :required="true"
      >
        <lfx-textarea
          v-model="form.auditScope"
          placeholder="Describe the areas of the codebase to be audited and the type of security review requested"
          :invalid="$v.auditScope.$error"
          @blur="$v.auditScope.$touch()"
        />
        <lfx-field-messages :validation="$v.auditScope" />
      </lfx-field>
    </template>

    <!-- Event-specific -->
    <template v-else-if="initiativeType === 'event'">
      <div class="grid grid-cols-2 gap-4">
        <lfx-field label="Event date">
          <lfx-input
            v-model="form.eventDate"
            type="date"
          />
        </lfx-field>

        <lfx-field label="Location">
          <lfx-input
            v-model="form.location"
            placeholder="e.g. San Francisco, CA or Virtual"
          />
        </lfx-field>
      </div>

      <lfx-field label="Eventbrite URL">
        <lfx-input
          v-model="form.eventbriteUrl"
          placeholder="https://eventbrite.com/e/..."
        />
      </lfx-field>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import useVuelidate from '@vuelidate/core';
import { required, url, helpers } from '@vuelidate/validators';
import type { InitiativeType } from './fundraise-step-type.vue';
import LfxField from '~/components/uikit/field/field.vue';
import LfxFieldMessages from '~/components/uikit/field/field-messages.vue';
import LfxInput from '~/components/uikit/input/input.vue';
import LfxTextarea from '~/components/uikit/textarea/textarea.vue';

export interface FundraiseDetailsForm {
  name: string;
  description: string;
  githubUrl: string;
  tags: string;
  auditScope: string;
  eventDate: string;
  location: string;
  eventbriteUrl: string;
}

const props = defineProps<{
  modelValue: FundraiseDetailsForm;
  initiativeType: InitiativeType | null;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: FundraiseDetailsForm): void;
}>();

const form = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val),
});

const rules = computed(() => {
  const base = {
    name: { required: helpers.withMessage('Initiative name is required', required) },
    description: { required: helpers.withMessage('Description is required', required) },
    githubUrl: {} as Record<string, unknown>,
    auditScope: {} as Record<string, unknown>,
  };

  if (props.initiativeType === 'security_audit') {
    base.githubUrl = {
      required: helpers.withMessage('GitHub URL is required', required),
      url: helpers.withMessage('Must be a valid URL', url),
    };
    base.auditScope = {
      required: helpers.withMessage('Audit scope is required', required),
    };
  } else if (props.initiativeType === 'project') {
    base.githubUrl = {
      url: helpers.withMessage('Must be a valid URL', url),
    };
  }

  return base;
});

const $v = useVuelidate(rules, form);

defineExpose({ $v });
</script>

<script lang="ts">
export default {
  name: 'FundraiseStepDetails',
};
</script>
