<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <lfx-popover
    placement="bottom-end"
    aria-label="User menu"
  >
    <lfx-avatar
      v-if="user"
      type="member"
      :src="user.avatarUrl"
      size="small"
      class="cursor-pointer"
    />
    <lfx-icon
      v-else
      name="circle-user"
      type="light"
      :size="20"
      class="cursor-pointer"
    />

    <template #content>
      <div class="c-dropdown w-60">
        <NuxtLink
          to="/my-donations"
          class="c-dropdown__item"
        >
          <lfx-icon
            name="circle-dollar-to-slot"
            type="light"
            :size="16"
          />
          My donations
        </NuxtLink>
        <NuxtLink
          to="/my-initiatives"
          class="c-dropdown__item"
        >
          <lfx-icon
            name="folder-heart"
            type="light"
            :size="16"
          />
          My initiatives
        </NuxtLink>
        <div class="c-dropdown__separator" />
        <div class="c-dropdown__item">
          <lfx-icon
            name="arrow-right-from-bracket"
            type="light"
            :size="16"
          />
          Sign out
        </div>
      </div>
    </template>
  </lfx-popover>
</template>

<script setup lang="ts">
import LfxAvatar from '~/components/uikit/avatar/avatar.vue';
import LfxIcon from '~/components/uikit/icon/icon.vue';
import LfxPopover from '~/components/uikit/popover/popover.vue';

// ---------------------------------------------------------------------------
// Mock auth — change MOCK_LOGGED_IN to true to preview the logged-in state
// ---------------------------------------------------------------------------
const MOCK_LOGGED_IN = false;
const mockUser = { name: 'Jane Doe', avatarUrl: 'https://i.pravatar.cc/100' };
const user = MOCK_LOGGED_IN ? mockUser : null;
// ---------------------------------------------------------------------------
</script>

<script lang="ts">
export default {
  name: 'CrowdfundingUserLogin',
};
</script>
