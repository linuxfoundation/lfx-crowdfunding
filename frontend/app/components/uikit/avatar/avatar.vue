<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <pv-avatar
    ref="avatarRef"
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
    @error="imageFailed = true"
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
const avatarRef = useTemplateRef<{ $el?: Element }>('avatarRef');

// A server-rendered <img> can fail before hydration attaches the error
// listener. naturalWidth is also 0 for a valid SVG with no intrinsic size,
// so confirm with decode(), which rejects only for a broken image.
onMounted(() => {
  const img = avatarRef.value?.$el?.querySelector?.('img');
  if (img?.complete && img.naturalWidth === 0) {
    const src = props.src;
    img.decode().catch(() => {
      if (props.src === src) {
        imageFailed.value = true;
      }
    });
  }
});

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
