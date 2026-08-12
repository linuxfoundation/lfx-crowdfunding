<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="flex flex-col gap-5">
    <div class="flex flex-col gap-1">
      <h2 class="text-base font-semibold text-neutral-900">Initiative attribution</h2>
      <p class="text-sm text-neutral-600 leading-5">Tell us who this initiative is being created on behalf of.</p>
    </div>

    <div class="border border-neutral-200 rounded-xl overflow-hidden">
      <div
        v-for="(option, index) in ATTRIBUTION_OPTIONS"
        :key="option.value"
      >
        <div
          class="flex flex-col gap-5 p-5"
          :class="[
            modelValue.kind === option.value ? 'bg-accent-50' : 'bg-white hover:bg-neutral-50',
            isDisabled(option.value) ? 'opacity-60 hover:bg-white' : '',
          ]"
        >
          <div
            class="flex gap-3"
            :class="isDisabled(option.value) ? 'cursor-not-allowed' : 'cursor-pointer'"
            @click="selectKind(option.value)"
          >
            <lfx-radio
              :model-value="modelValue.kind"
              :value="option.value"
              :disabled="isDisabled(option.value)"
              @update:model-value="selectKind($event as AttributionKind)"
            />
            <div class="flex flex-col gap-1">
              <span class="text-sm font-semibold text-neutral-900">{{ option.label }}</span>
              <p class="text-xs text-neutral-600 leading-4">{{ option.description }}</p>
              <p
                v-if="isDisabled(option.value)"
                class="text-xs text-neutral-500 leading-4 italic"
              >
                {{ option.emptyMessage }}
              </p>
            </div>
          </div>

          <!-- Entity picker — only for non-personal kinds, revealed when selected -->
          <div
            v-if="option.value !== 'personal' && modelValue.kind === option.value && !isDisabled(option.value)"
            class="flex flex-col gap-2 pl-7"
          >
            <p class="text-sm font-medium text-neutral-900">
              {{ option.entityLabel }} <span class="text-negative-500">*</span>
            </p>

            <lfx-select
              :model-value="modelValue.entityId ?? ''"
              placeholder="Select option..."
              @update:model-value="selectEntity($event)"
            >
              <lfx-dropdown-item
                v-for="entity in candidatesFor(option.value)"
                :key="entity.id"
                :value="entity.id"
                :label="entity.name"
              />
            </lfx-select>

            <p class="text-xs text-neutral-600 flex items-center gap-1">
              <lfx-icon
                name="circle-info"
                :size="12"
              />
              {{ option.escapeHatchLabel }}
              <a
                :href="affiliationsManagementUrl"
                target="_blank"
                rel="noopener noreferrer"
                class="text-accent-500 underline hover:text-accent-600"
                >Work History &amp; Affiliations</a
              >.
            </p>
          </div>
        </div>

        <div
          v-if="index < ATTRIBUTION_OPTIONS.length - 1"
          class="border-b border-neutral-200"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { ATTRIBUTION_OPTIONS, AFFILIATIONS_MANAGEMENT_PATH } from '../../config/attribution.config';
import LfxRadio from '~/components/uikit/radio/radio.vue';
import LfxSelect from '~/components/uikit/select/select.vue';
import LfxDropdownItem from '~/components/uikit/dropdown/dropdown-item.vue';
import LfxIcon from '~/components/uikit/icon/icon.vue';
import type { AttributionData, AttributionKind } from '~/types/fundraise.types';
import type { AffiliationEntity } from '#shared/types/affiliation.types';

const props = defineProps<{
  modelValue: AttributionData;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: AttributionData): void;
}>();

// Fetched once per drawer session — a single-select list is small enough that
// Nuxt's own request de-dupe/caching covers this without a dedicated composable.
const { data } = useLazyFetch('/api/me/affiliations');

const candidatesFor = (kind: AttributionKind): AffiliationEntity[] => {
  if (kind === 'organization') return data.value?.organizations ?? [];
  if (kind === 'project') return data.value?.projects ?? [];
  return [];
};

// Personal is always available; org/project are disabled once the fetch has
// resolved with an empty candidate list. Before it resolves, data.value is
// undefined, so `candidatesFor` returns [] transiently — the row starts
// disabled and enables itself once the fetch confirms there's nothing to pick
// from either way, so this never flickers enabled→disabled.
const isDisabled = (kind: AttributionKind): boolean => kind !== 'personal' && candidatesFor(kind).length === 0;

const affiliationsManagementUrl = computed(
  () => `${useRuntimeConfig().public.selfServeUrl}${AFFILIATIONS_MANAGEMENT_PATH}`,
);

const selectKind = (kind: AttributionKind) => {
  if (isDisabled(kind)) return;
  if (kind === 'personal') {
    emit('update:modelValue', { kind, entityId: null });
    return;
  }
  emit('update:modelValue', { kind, entityId: props.modelValue.kind === kind ? props.modelValue.entityId : null });
};

const selectEntity = (entityId: string) => {
  emit('update:modelValue', { ...props.modelValue, entityId: entityId || null });
};
</script>

<script lang="ts">
export default {
  name: 'FundraiseAttributionStep',
};
</script>
