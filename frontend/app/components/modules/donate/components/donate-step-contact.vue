<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="flex flex-col gap-3 mb-4">
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

      <div class="h-px bg-neutral-200" />

      <!-- Organization row: shows the picked org once there is one -->
      <div class="flex flex-col gap-4 p-6 w-full">
        <div
          class="flex gap-3 items-center w-full cursor-pointer"
          @click="selectedKey = 'organization'"
        >
          <lfx-avatar
            type="organization"
            size="normal"
          />
          <div class="flex-1 min-w-0">
            <p class="text-sm font-semibold text-neutral-900 truncate">
              {{ form.organization?.name ?? 'An organization' }}
            </p>
            <p class="text-xs text-neutral-600 truncate">
              {{ form.organization?.domain ?? 'Donate on behalf of a company or organization' }}
            </p>
          </div>
          <lfx-radio
            v-model="selectedKey"
            value="organization"
            name="donor-type"
          />
        </div>

        <template v-if="form.donorType === 'organization'">
          <!-- Search (Clearbit) -->
          <div
            v-if="!manualEntry"
            class="flex flex-col gap-2"
          >
            <lfx-select
              :model-value="form.organization?.domain ?? ''"
              :placeholder="form.organization ? 'Search for a different organization' : 'Search organizations'"
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
          </div>

          <!-- Typed by hand -->
          <div
            v-else
            class="flex flex-col gap-3"
          >
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
          </div>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, watch, computed } from 'vue';
import LfxAvatar from '~/components/uikit/avatar/avatar.vue';
import LfxDropdownItem from '~/components/uikit/dropdown/dropdown-item.vue';
import LfxDropdownSearch from '~/components/uikit/dropdown/dropdown-search.vue';
import LfxField from '~/components/uikit/field/field.vue';
import LfxFieldMessage from '~/components/uikit/field/field-message.vue';
import LfxInput from '~/components/uikit/input/input.vue';
import LfxLinkButton from '~/components/uikit/link-button/link-button.vue';
import LfxRadio from '~/components/uikit/radio/radio.vue';
import LfxSelect from '~/components/uikit/select/select.vue';
import LfxSpinner from '~/components/uikit/spinner/spinner.vue';
import { useOrgSuggest } from '~/composables/useOrgSuggest';
import {
  ORG_SEARCH_MIN_CHARS,
  ORG_WEBSITE_PATTERN,
  manualOrgClaim,
} from '~/components/modules/donate/config/org-search.config';
import type { DonateContactForm } from '#shared/types/donate.types';

const props = defineProps<{
  modelValue: DonateContactForm;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: DonateContactForm): void;
}>();

const { user } = useAuth();

const form = reactive<DonateContactForm>({ ...props.modelValue });

const query = ref('');
const { data: suggestions, isFetching, isError, enabled } = useOrgSuggest(query);

const manualEntry = ref(false);
const manualName = ref('');
const manualWebsite = ref('');

const websiteInvalid = computed(
  () => manualWebsite.value.trim() !== '' && !ORG_WEBSITE_PATTERN.test(manualWebsite.value.trim()),
);

watch(
  () => props.modelValue,
  (val) => Object.assign(form, val),
  { deep: true },
);

watch(form, (val) => emit('update:modelValue', { ...val }), { deep: true });

const selectedKey = computed({
  get: () => form.donorType,
  set: (val: string | number | boolean) => {
    form.donorType = val === 'organization' ? 'organization' : 'individual';
  },
});

const pickSuggestion = (domain: string) => {
  const org = suggestions.value?.find((s) => s.domain === domain);
  if (org) form.organization = { ...org };
};

watch([manualName, manualWebsite], ([name, website]) => {
  if (manualEntry.value) form.organization = manualOrgClaim(name, website);
});

const startManualEntry = () => {
  manualEntry.value = true;
  manualName.value = query.value.trim();
  manualWebsite.value = '';
  form.organization = null;
};

const stopManualEntry = () => {
  manualEntry.value = false;
  form.organization = null;
};
</script>

<script lang="ts">
export default {
  name: 'DonateStepContact',
};
</script>
