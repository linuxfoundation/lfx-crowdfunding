<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <lfx-button
    label="Donate"
    type="primary"
    button-style="pill"
    icon="hand-heart"
    v-bind="$attrs"
    @click="handleClick()"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue';
import LfxButton from '~/components/uikit/button/button.vue';
import { useDonateDrawerStore } from '~/components/modules/donate/store/donate-drawer.store';
import { useAuth } from '~/composables/useAuth';
import { useInitiative } from '~/composables/initiatives/useInitiative';
import { AppRoute } from '~/config/routes';

const route = useRoute();
const router = useRouter();

const isInitiativeDetailPage = computed(() => /^\/initiatives\/[^/]+$/.test(route.path));
const initiativeId = computed(() => (isInitiativeDetailPage.value ? (route.params.slug as string) : ''));

const { data: initiative } = useInitiative(initiativeId);
const { openDonateDrawer } = useDonateDrawerStore();
const { isAuthenticated, login } = useAuth();

function handleClick() {
  if (!isInitiativeDetailPage.value) {
    router.push(AppRoute.Initiatives);
    return;
  }
  if (!isAuthenticated.value) {
    login();
    return;
  }
  if (initiative.value) {
    openDonateDrawer({
      id: initiative.value.id,
      name: initiative.value.name,
      logoUrl: initiative.value.logoUrl,
    });
  }
}
</script>

<script lang="ts">
export default {
  name: 'DonateButton',
  inheritAttrs: false,
};
</script>
