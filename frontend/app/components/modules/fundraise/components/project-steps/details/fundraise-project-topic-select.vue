<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div
    ref="containerRef"
    class="relative"
  >
    <!-- Trigger -->
    <button
      type="button"
      class="w-full border border-neutral-200 rounded-full bg-white shadow-xs flex items-center justify-between min-h-9 px-3 py-1 gap-2 text-left text-sm transition-all focus:outline-none"
      :class="{ 'border-neutral-900': isOpen }"
      @click="isOpen = !isOpen"
    >
      <div class="flex flex-wrap gap-1.5 flex-1 min-w-0">
        <span
          v-if="modelValue.length === 0"
          class="text-sm text-neutral-400 leading-5"
          >Select topic(s)</span
        >
        <span
          v-for="topic in selectedTopics"
          :key="topic.value"
          class="flex items-center gap-1 text-xs font-medium text-neutral-900 bg-neutral-100 rounded-full pl-2.5 pr-1.5 py-0.5 shrink-0"
        >
          {{ topic.label }}
          <button
            type="button"
            class="text-neutral-500 hover:text-neutral-900 leading-none"
            @click.stop="remove(topic.value)"
          >
            <lfx-icon
              name="xmark"
              type="solid"
              :size="10"
            />
          </button>
        </span>
      </div>
      <lfx-icon
        name="angle-down"
        type="light"
        :size="12"
        class="shrink-0 text-neutral-900 transition-transform"
        :class="{ 'rotate-180': isOpen }"
      />
    </button>

    <!-- Dropdown panel -->
    <div
      v-if="isOpen"
      class="absolute top-full mt-1 left-0 right-0 z-50 bg-white border border-neutral-200 rounded-xl shadow-lg py-1 max-h-52 overflow-y-auto"
    >
      <button
        v-for="option in OPTIONS"
        :key="option.value"
        type="button"
        class="w-full px-3 py-2 text-sm text-left flex items-center gap-2.5 hover:bg-neutral-50 transition-colors"
        @click="toggle(option.value)"
      >
        <div
          class="size-4 shrink-0 rounded border flex items-center justify-center transition-colors"
          :class="isSelected(option.value) ? 'bg-accent-500 border-accent-500' : 'border-neutral-300 bg-white'"
        >
          <lfx-icon
            v-if="isSelected(option.value)"
            name="check"
            type="solid"
            :size="9"
            class="text-white"
          />
        </div>
        <span class="text-neutral-900">{{ option.label }}</span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import { onClickOutside } from '@vueuse/core';
import LfxIcon from '~/components/uikit/icon/icon.vue';

interface TopicOption {
  value: string;
  label: string;
}

const OPTIONS: TopicOption[] = [
  { value: 'security', label: 'Security' },
  { value: 'cloud_native', label: 'Cloud Native' },
  { value: 'developer_tools', label: 'Developer Tools' },
  { value: 'ai_ml', label: 'AI / ML' },
  { value: 'infrastructure', label: 'Infrastructure' },
  { value: 'devops', label: 'DevOps' },
  { value: 'observability', label: 'Observability' },
  { value: 'networking', label: 'Networking' },
  { value: 'storage', label: 'Storage' },
  { value: 'serverless', label: 'Serverless' },
  { value: 'web_standards', label: 'Web Standards' },
  { value: 'runtime', label: 'Runtime' },
];

const props = defineProps<{
  modelValue: string[];
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: string[]): void;
}>();

const containerRef = ref<HTMLElement | null>(null);
const isOpen = ref(false);

onClickOutside(containerRef, () => {
  isOpen.value = false;
});

const selectedTopics = computed(() => OPTIONS.filter((o) => props.modelValue.includes(o.value)));

const isSelected = (value: string) => props.modelValue.includes(value);

const toggle = (value: string) => {
  const next = isSelected(value) ? props.modelValue.filter((v) => v !== value) : [...props.modelValue, value];
  emit('update:modelValue', next);
};

const remove = (value: string) => {
  emit(
    'update:modelValue',
    props.modelValue.filter((v) => v !== value),
  );
};
</script>

<script lang="ts">
export default {
  name: 'FundraiseProjectTopicSelect',
};
</script>
