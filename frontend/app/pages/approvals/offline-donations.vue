<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <offline-donations />
</template>

<script setup lang="ts">
import OfflineDonations from '~/components/modules/offline-donations/view/offline-donations.vue';

definePageMeta({ middleware: 'auth' });

useHead({ title: 'Record an offline donation' });
</script>
