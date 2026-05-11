<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="min-h-screen flex flex-col">
    <crowdfunding-header />
    <main class="flex-grow">
      <slot />
    </main>
    <crowdfunding-footer />
  </div>
</template>

<script setup lang="ts">
import CrowdfundingHeader from '~/components/shared/layout/header.vue';
import CrowdfundingFooter from '~/components/shared/layout/footer.vue';
</script>
