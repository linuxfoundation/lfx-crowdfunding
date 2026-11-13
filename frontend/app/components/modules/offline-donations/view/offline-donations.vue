<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="container py-10 flex flex-col gap-8 max-w-3xl">
    <div class="flex flex-col gap-2">
      <NuxtLink
        v-if="initiative"
        :to="`/initiatives/${initiative.slug}`"
        class="self-start"
      >
        <lfx-link-button
          :label="`Back to ${initiative.name}`"
          icon="arrow-left"
          variant="accent"
        />
      </NuxtLink>
      <h1 class="text-2xl font-semibold text-neutral-900">Record an offline donation</h1>
      <p class="text-sm text-neutral-600">
        For organization donations LF finance received by invoice or wire. The donation is added to the initiative's
        Ledger balance and donor list, and to the organization's donations on its LFX organization page.
      </p>
    </div>

    <div
      v-if="approverLoading"
      class="flex justify-center py-16"
    >
      <lfx-spinner :size="32" />
    </div>

    <lfx-card
      v-else-if="!isApprover"
      class="p-8 flex flex-col gap-3 text-center"
    >
      <lfx-icon
        name="lock"
        type="regular"
        :size="32"
        class="text-neutral-400 mx-auto"
      />
      <p class="text-sm text-neutral-700">Only approvers can record offline donations.</p>
    </lfx-card>

    <template v-else>
      <!-- Initiative -->
      <lfx-card class="p-6 flex flex-col gap-3">
        <p class="text-sm font-semibold text-neutral-900">Initiative</p>
        <div
          v-if="initiativeLoading"
          class="flex justify-center py-4"
        >
          <lfx-spinner />
        </div>
        <div
          v-else-if="initiative"
          class="flex items-center gap-3"
        >
          <lfx-avatar
            type="organization"
            size="normal"
            :src="initiative.logoUrl"
          />
          <div class="flex-1 min-w-0">
            <p class="text-sm font-semibold text-neutral-900 truncate">{{ initiative.name }}</p>
            <p class="text-xs text-neutral-600 truncate">{{ initiative.slug }}</p>
          </div>
          <lfx-link-button
            label="Change"
            variant="accent"
            @click="selectInitiative('')"
          />
        </div>
        <template v-else>
          <lfx-select
            model-value=""
            placeholder="Search published initiatives"
            @update:model-value="selectInitiative"
          >
            <lfx-dropdown-search
              v-model="initiativeQuery"
              placeholder="Initiative name"
              autofocus
              lazy
            />
            <div
              v-if="searchFetching && !searchResults.length"
              class="flex justify-center px-3 py-2"
            >
              <lfx-spinner />
            </div>
            <div
              v-else-if="!searchResults.length"
              class="px-3 py-2 text-xs text-neutral-500"
            >
              No initiatives found
            </div>
            <lfx-dropdown-item
              v-for="item in searchResults"
              :key="item.slug"
              :value="item.slug"
              :label="item.name"
            />
          </lfx-select>
          <p
            v-if="initiativeError"
            class="text-xs text-negative-600"
          >
            That initiative wasn't found. Search for it instead.
          </p>
        </template>
      </lfx-card>

      <offline-donation-form
        v-if="initiative"
        :key="initiative.id"
        :initiative="initiative"
        @recorded="onRecorded"
      />

      <offline-donation-list
        v-if="initiative"
        :initiative-id="initiative.id"
      />
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { useQueryClient } from '@tanstack/vue-query';
import OfflineDonationForm from '../components/offline-donation-form.vue';
import OfflineDonationList from '../components/offline-donation-list.vue';
import { OFFLINE_DONATIONS_PATH } from '../config/offline-donation.config';
import LfxAvatar from '~/components/uikit/avatar/avatar.vue';
import LfxCard from '~/components/uikit/card/card.vue';
import LfxDropdownItem from '~/components/uikit/dropdown/dropdown-item.vue';
import LfxDropdownSearch from '~/components/uikit/dropdown/dropdown-search.vue';
import LfxIcon from '~/components/uikit/icon/icon.vue';
import LfxLinkButton from '~/components/uikit/link-button/link-button.vue';
import LfxSelect from '~/components/uikit/select/select.vue';
import LfxSpinner from '~/components/uikit/spinner/spinner.vue';
import { useInitiative } from '~/composables/initiatives/useInitiative';
import { useInitiatives } from '~/composables/initiatives/useInitiatives';
import { useIsApprover } from '~/composables/useIsApprover';

const route = useRoute();
const router = useRouter();
const queryClient = useQueryClient();

const { isApprover, isLoading: approverLoading } = useIsApprover();

const slug = computed(() => (typeof route.query.initiative === 'string' ? route.query.initiative : ''));
const { data: initiative, isLoading: initiativeLoading, error: initiativeError } = useInitiative(slug);

const initiativeQuery = ref('');
const { data: searchData, isFetching: searchFetching } = useInitiatives({ search: initiativeQuery, pageSize: 10 });
const searchResults = computed(() => searchData.value?.pages[0]?.data ?? []);

const selectInitiative = (value: string) => {
  router.replace({ path: OFFLINE_DONATIONS_PATH, query: value ? { initiative: value } : {} });
};

const onRecorded = () => {
  queryClient.invalidateQueries({ queryKey: ['offline-donations'] });
};
</script>

<script lang="ts">
export default {
  name: 'OfflineDonations',
};
</script>
