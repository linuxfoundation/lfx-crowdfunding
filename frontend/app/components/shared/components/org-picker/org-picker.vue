<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="flex flex-col gap-2">
    <!-- Search (Clearbit), as in the donate drawer (#409) -->
    <template v-if="!manualEntry">
      <lfx-select
        :model-value="modelValue?.domain ?? ''"
        :placeholder="modelValue ? 'Search for a different organization' : 'Search organizations'"
        @update:model-value="pickSuggestion"
      >
        <lfx-dropdown-search
          v-model="query"
          placeholder="Company name"
          autofocus
        />
        <div
          v-if="!enabled"
          class="px-3 py-2 text-xs text-neutral-500"
        >
          Type at least {{ ORG_SEARCH_MIN_CHARS }} characters
        </div>
        <div
          v-else-if="isFetching && !suggestions?.length"
          class="flex justify-center px-3 py-2"
        >
          <lfx-spinner />
        </div>
        <div
          v-else-if="isError"
          class="px-3 py-2 text-xs text-neutral-500"
        >
          Search isn't available right now. Enter the organization manually.
        </div>
        <div
          v-else-if="!suggestions?.length"
          class="px-3 py-2 text-xs text-neutral-500"
        >
          No organizations found
        </div>
        <lfx-dropdown-item
          v-for="org in suggestions"
          :key="org.domain"
          :value="org.domain"
          :label="org.name"
        >
          <span class="flex items-center gap-2 min-w-0">
            <lfx-avatar
              type="organization"
              size="xsmall"
              :src="org.logoUrl"
            />
            <span class="truncate">{{ org.name }}</span>
            <span class="text-xs text-neutral-500 truncate">{{ org.domain }}</span>
          </span>
        </lfx-dropdown-item>
      </lfx-select>
      <lfx-link-button
        label="Can't find it? Enter it manually"
        variant="accent"
        class="self-start"
        @click="startManualEntry"
      />
    </template>

    <!-- Typed by hand -->
    <template v-else>
      <lfx-field
        label="Organization name"
        required
      >
        <lfx-input
          v-model="manualName"
          aria-label="Organization name"
          placeholder="Acme Inc."
        />
      </lfx-field>
      <lfx-field
        label="Website"
        required
      >
        <lfx-input
          v-model="manualWebsite"
          aria-label="Website"
          placeholder="acme.com"
          :invalid="websiteInvalid"
        />
        <lfx-field-message
          v-if="websiteInvalid"
          type="error"
        >
          Enter the organization's website, e.g. acme.com
        </lfx-field-message>
      </lfx-field>
      <lfx-link-button
        label="Search instead"
        variant="accent"
        class="self-start"
        @click="stopManualEntry"
      />
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import LfxAvatar from '~/components/uikit/avatar/avatar.vue';
import LfxDropdownItem from '~/components/uikit/dropdown/dropdown-item.vue';
import LfxDropdownSearch from '~/components/uikit/dropdown/dropdown-search.vue';
import LfxField from '~/components/uikit/field/field.vue';
import LfxFieldMessage from '~/components/uikit/field/field-message.vue';
import LfxInput from '~/components/uikit/input/input.vue';
import LfxLinkButton from '~/components/uikit/link-button/link-button.vue';
import LfxSelect from '~/components/uikit/select/select.vue';
import LfxSpinner from '~/components/uikit/spinner/spinner.vue';
import { useOrgSuggest } from '~/composables/useOrgSuggest';
import {
  ORG_SEARCH_MIN_CHARS,
  ORG_WEBSITE_PATTERN,
  manualOrgClaim,
} from '~/components/modules/donate/config/org-search.config';
import type { OrgClaim } from '#shared/types/organization.types';

defineProps<{ modelValue: OrgClaim | null }>();

const emit = defineEmits<{ (e: 'update:modelValue', value: OrgClaim | null): void }>();

const query = ref('');
const { data: suggestions, isFetching, isError, enabled } = useOrgSuggest(query);

const manualEntry = ref(false);
const manualName = ref('');
const manualWebsite = ref('');

const websiteInvalid = computed(
  () => manualWebsite.value.trim() !== '' && !ORG_WEBSITE_PATTERN.test(manualWebsite.value.trim()),
);

const pickSuggestion = (domain: string) => {
  const org = suggestions.value?.find((s) => s.domain === domain);
  if (org) emit('update:modelValue', { name: org.name, domain: org.domain });
};

watch([manualName, manualWebsite], ([name, website]) => {
  if (manualEntry.value) emit('update:modelValue', manualOrgClaim(name, website));
});

const startManualEntry = () => {
  manualEntry.value = true;
  manualName.value = query.value.trim();
  manualWebsite.value = '';
  emit('update:modelValue', null);
};

const stopManualEntry = () => {
  manualEntry.value = false;
  emit('update:modelValue', null);
};
</script>

<script lang="ts">
export default {
  name: 'OrgPicker',
};
</script>
