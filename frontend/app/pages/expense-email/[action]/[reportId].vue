<!--
  Copyright The Linux Foundation and each contributor to LFX.
  SPDX-License-Identifier: MIT
-->
<template>
  <div class="flex flex-col items-center justify-center min-h-screen gap-4">
    <lfx-spinner :size="32" />
    <p class="text-body-1 text-neutral-500">Redirecting&hellip;</p>
  </div>
</template>

<script setup lang="ts">
import LfxSpinner from '~/components/uikit/spinner/spinner.vue';
import useToastService from '~/components/uikit/toast/toast.service';
import { ToastTypesEnum } from '~/components/uikit/toast/types/toast.types';

// Require authentication — if the user is not logged in they will be redirected
// to Auth0 and returned here after login.
definePageMeta({ middleware: ['auth'] });

useHead({ title: 'Processing expense action…' });

const route = useRoute();
const router = useRouter();
const { showToast } = useToastService();

const action = route.params.action as string;
const reportId = route.params.reportId as string;

const ACTION_LABELS: Record<string, { verb: string; past: string }> = {
  approve: { verb: 'approve', past: 'approved' },
  reject: { verb: 'reject', past: 'rejected' },
};

onMounted(async () => {
  const label = ACTION_LABELS[action];

  if (!label) {
    showToast(`Unknown action "${action}". Please use the link from your email.`, ToastTypesEnum.negative);
    await router.replace('/');
    return;
  }

  try {
    await $fetch(`/api/expense-email/${encodeURIComponent(action)}/${encodeURIComponent(reportId)}`, {
      method: 'POST',
    });
    showToast(`The expense report has been ${label.past}.`, ToastTypesEnum.positive);
  } catch {
    showToast(
      `Failed to ${label.verb} the expense report. Please contact LF Support.`,
      ToastTypesEnum.negative,
    );
  }

  await router.replace('/');
});
</script>
