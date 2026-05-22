<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="flex justify-center items-center gap-6 py-8">
    <lfx-tooltip
      content="Share on X"
      @click="twitter()"
    >
      <div
        class="cursor-pointer flex items-center justify-center w-10 h-10 rounded-full bg-neutral-100 hover:bg-neutral-200 transition-colors"
      >
        <lfx-icon
          name="x-twitter"
          type="brands"
          :size="20"
        />
      </div>
    </lfx-tooltip>

    <lfx-tooltip
      content="Share on LinkedIn"
      @click="linkedin()"
    >
      <div
        class="cursor-pointer flex items-center justify-center w-10 h-10 rounded-full bg-neutral-100 hover:bg-neutral-200 transition-colors"
      >
        <lfx-icon
          name="linkedin"
          type="brands"
          :size="20"
        />
      </div>
    </lfx-tooltip>

    <lfx-tooltip
      content="Share on Reddit"
      @click="reddit()"
    >
      <div
        class="cursor-pointer flex items-center justify-center w-10 h-10 rounded-full bg-neutral-100 hover:bg-neutral-200 transition-colors"
      >
        <lfx-icon
          name="reddit"
          type="brands"
          :size="20"
        />
      </div>
    </lfx-tooltip>

    <lfx-tooltip
      content="Send email"
      @click="email()"
    >
      <lfx-icon-button
        icon="envelope"
        size="large"
      />
    </lfx-tooltip>
  </div>

  <div class="flex items-center">
    <div class="border-t border-neutral-200 flex-grow" />
    <p class="text-body-2 uppercase text-neutral-500 px-4">or</p>
    <div class="border-t border-neutral-200 flex-grow" />
  </div>

  <div class="flex items-center pt-8 gap-3">
    <lfx-button
      type="tertiary"
      class="!rounded-full w-full flex justify-center"
      @click="copy()"
    >
      <lfx-icon name="clone" />
      Copy link
    </lfx-button>
  </div>
</template>

<script setup lang="ts">
import LfxTooltip from '~/components/uikit/tooltip/tooltip.vue';
import LfxIcon from '~/components/uikit/icon/icon.vue';
import LfxIconButton from '~/components/uikit/icon-button/icon-button.vue';
import LfxButton from '~/components/uikit/button/button.vue';
import { ToastTypesEnum } from '~/components/uikit/toast/types/toast.types';
import useToastService from '~/components/uikit/toast/toast.service';
import type { ShareData } from '~/types/share.types';

const props = defineProps<{
  defaults: ShareData;
}>();

const emit = defineEmits<{ (e: 'copied'): void }>();

const { showToast } = useToastService();

const copy = () => {
  navigator?.clipboard.writeText(props.defaults.url);
  showToast('Link copied to clipboard', ToastTypesEnum.positive);
  emit('copied');
};

const email = () => {
  const title = props.defaults?.title ? `Check this out: ${props.defaults.title}` : 'Check this out';
  const url = encodeURIComponent(props.defaults.url);
  window?.open(`mailto:?subject=${title}&body=${url}`, '_blank');
};

const twitter = () => {
  const { url } = props.defaults;
  const title = props.defaults?.title ? `Explore ${props.defaults.title}` : 'Explore this';
  const link = `https://twitter.com/intent/tweet?text=${encodeURIComponent(`${title} ${url}`)}`;
  const width = 640;
  const height = 480;
  const left = window.screen.width / 2 - width / 2;
  const top = window.screen.height / 2 - height / 2;
  window?.open(
    link,
    '_blank',
    `width=${width},height=${height},top=${top},left=${left},menubar=no,location=no,status=no`,
  );
};

const reddit = () => {
  const url = encodeURIComponent(props.defaults.url);
  const title = props.defaults?.title ? `Explore ${props.defaults.title}` : 'Explore this';
  window?.open(`https://www.reddit.com/submit?title=${title}&url=${url}`, '_blank');
};

const linkedin = () => {
  const url = encodeURIComponent(props.defaults.url);
  const link = `https://www.linkedin.com/sharing/share-offsite/?url=${url}`;
  const width = 640;
  const height = 480;
  const left = window.screen.width / 2 - width / 2;
  const top = window.screen.height / 2 - height / 2;
  window?.open(
    link,
    '_blank',
    `width=${width},height=${height},top=${top},left=${left},menubar=no,location=no,status=no`,
  );
};
</script>

<script lang="ts">
export default {
  name: 'ShareActions',
};
</script>
