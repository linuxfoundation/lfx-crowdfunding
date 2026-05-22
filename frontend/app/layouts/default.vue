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
    <donate-drawer-global />
    <fundraise-drawer-global />
    <share-modal-global />
  </div>
</template>

<script setup lang="ts">
import CrowdfundingHeader from '~/components/shared/layout/header.vue';
import CrowdfundingFooter from '~/components/shared/layout/footer.vue';
import DonateDrawerGlobal from '~/components/modules/donate/components/donate-drawer-global.vue';
import FundraiseDrawerGlobal from '~/components/modules/fundraise/components/fundraise-drawer-global.vue';
import ShareModalGlobal from '~/components/shared/components/share/share-modal-global.vue';
</script>
