<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <client-only>
    <lfx-popover
      v-if="isAuthenticated"
      placement="bottom-end"
      aria-label="User menu"
    >
      <lfx-avatar
        type="member"
        :src="user?.picture"
        size="small"
        class="cursor-pointer"
      />

      <template #content>
        <div class="c-dropdown w-60">
          <a
            :href="`${selfServeUrl}/crowdfunding/donations`"
            target="_blank"
            rel="noopener noreferrer"
            class="c-dropdown__item"
          >
            <lfx-icon
              name="circle-dollar-to-slot"
              type="light"
              :size="16"
            />
            My donations
          </a>
          <a
            :href="`${selfServeUrl}/crowdfunding/initiatives`"
            target="_blank"
            rel="noopener noreferrer"
            class="c-dropdown__item"
          >
            <lfx-icon
              name="folder-heart"
              type="light"
              :size="16"
            />
            My initiatives
          </a>
          <div class="c-dropdown__separator" />
          <button
            class="c-dropdown__item w-full text-left"
            @click="logout"
          >
            <lfx-icon
              name="arrow-right-from-bracket"
              type="light"
              :size="16"
            />
            Sign out
          </button>
        </div>
      </template>
    </lfx-popover>

    <lfx-icon-button
      v-else
      icon="circle-user"
      type="transparent"
      size="medium"
      :loading="isLoading"
      aria-label="Sign in"
      @click="login()"
    />

    <!-- SSR/hydration fallback: matches the signed-out button so anonymous
         users (the common case) see no flash. Non-interactive placeholder;
         auth state only resolves client-side (LFXV2-2700). -->
    <template #fallback>
      <lfx-icon-button
        icon="circle-user"
        type="transparent"
        size="medium"
        aria-label="Sign in"
      />
    </template>
  </client-only>
</template>

<script setup lang="ts">
import LfxAvatar from '~/components/uikit/avatar/avatar.vue';
import LfxIconButton from '~/components/uikit/icon-button/icon-button.vue';
import LfxIcon from '~/components/uikit/icon/icon.vue';
import LfxPopover from '~/components/uikit/popover/popover.vue';
import { useAuth } from '~/composables/useAuth';

const { isAuthenticated, user, isLoading, login, logout } = useAuth();

const {
  public: { selfServeUrl },
} = useRuntimeConfig();
</script>

<script lang="ts">
export default {
  name: 'CrowdfundingUserLogin',
};
</script>
