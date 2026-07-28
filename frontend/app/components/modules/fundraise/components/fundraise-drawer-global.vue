<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <fundraise-drawer
    v-if="isOpen"
    v-model="isOpen"
  />
</template>

<script setup lang="ts">
import { onMounted } from 'vue';
import { storeToRefs } from 'pinia';
import { useRoute, useRouter } from 'nuxt/app';
import { useFundraiseDrawerStore } from '../store/fundraise-drawer.store';
import FundraiseDrawer from './fundraise-drawer.vue';
import { GITHUB_FUNDRAISE_SESSION_KEY } from '~/composables/useGithubAuth';

const fundraiseDrawerStore = useFundraiseDrawerStore();
const { isOpen } = storeToRefs(fundraiseDrawerStore);
const route = useRoute();
const router = useRouter();

onMounted(async () => {
  if (route.query.github_connected === 'true' && sessionStorage.getItem(GITHUB_FUNDRAISE_SESSION_KEY)) {
    fundraiseDrawerStore.openFundraiseDrawer();
  } else if (route.query.fundraise === 'true') {
    await router.replace({ query: { ...route.query, fundraise: undefined } });
    fundraiseDrawerStore.openFundraiseDrawer();
  }
});
</script>

<script lang="ts">
export default {
  name: 'FundraiseDrawerGlobal',
};
</script>
