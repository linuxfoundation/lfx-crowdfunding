<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <initiative-process-approval
    :slug="slug"
    :action="action"
  />
</template>

<script setup lang="ts">
import InitiativeProcessApproval from '~/components/modules/initiatives/process-approval/process-approval.vue';

definePageMeta({ middleware: 'auth' });

const route = useRoute();
const slug = route.params.slug as string;
const action = route.params.action as string;

useHead({ title: `${action === 'approve' ? 'Approve' : 'Decline'} Initiative` });
</script>
