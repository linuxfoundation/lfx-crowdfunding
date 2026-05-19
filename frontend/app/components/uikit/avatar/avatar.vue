<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <pv-avatar
    :icon="icon"
    :image="imageFailed ? undefined : props.src"
    :shape="props.type === 'organization' ? 'square' : 'circle'"
    :size="props.size"
    :class="{
      [`type-${props.type}`]: true,
      'p-avatar-sm': props.size === 'small',
      'p-avatar-xsmall': props.size === 'xsmall',
      'has-image': props.src && !imageFailed,
    }"
    v-bind="$attrs"
    @image-error="imageFailed = true"
  />
</template>

<script setup lang="ts">
import type { AvatarSize, AvatarType } from './types/Avatar.types';
import { AvatarIcons } from './types/Avatar.types';

const props = withDefaults(
  defineProps<{
    type: AvatarType;
    size?: AvatarSize;
    src?: string;
  }>(),
  {
    size: 'normal',
    type: 'member',
    src: undefined,
  },
);

const imageFailed = ref(false);

watch(
  () => props.src,
  () => {
    imageFailed.value = false;
  },
);

const icon = computed(() => {
  if (props.src && !imageFailed.value) {
    return undefined;
  }

  if (props.type === 'project') {
    return AvatarIcons.Project;
  }

  return props.type === 'member' ? AvatarIcons.Member : AvatarIcons.Organization;
});
</script>

<script lang="ts">
export default {
  name: 'LfxAvatar',
};
</script>
