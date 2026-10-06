<!--
  Copyright The Linux Foundation and each contributor to LFX.
  SPDX-License-Identifier: MIT
-->
<template>
  <div class="flex flex-col items-center justify-center min-h-screen gap-4 px-4">
    <lfx-card
      v-if="config"
      class="flex flex-col gap-4 p-8 max-w-md w-full"
    >
      <h1 class="text-heading-3 font-semibold">{{ config.title }}</h1>
      <p class="text-body-1 text-neutral-500">
        Expense report <strong>#{{ reportId }}</strong>
      </p>
      <p class="text-body-1 text-neutral-500">{{ config.description }} This action cannot be undone.</p>
      <div class="flex gap-3 justify-end">
        <lfx-button
          type="secondary"
          label="Cancel"
          :disabled="submitting"
          @click="router.replace('/')"
        />
        <lfx-button
          type="primary"
          :label="config.confirmLabel"
          :loading="submitting"
          @click="confirm"
        />
      </div>
    </lfx-card>
    <lfx-spinner
      v-else
      :size="32"
    />
  </div>
</template>

<script setup lang="ts">
import LfxButton from '~/components/uikit/button/button.vue';
import LfxCard from '~/components/uikit/card/card.vue';
import LfxSpinner from '~/components/uikit/spinner/spinner.vue';
import useToastService from '~/components/uikit/toast/toast.service';
import { ToastTypesEnum } from '~/components/uikit/toast/types/toast.types';
import { getExpenseAction } from '~/components/modules/expense-email/config/expense-action.config';

// Require authentication — if the user is not logged in they will be redirected
// to Auth0 and returned here after login.
definePageMeta({ middleware: ['auth'] });

useHead({ title: 'Confirm expense action' });

const route = useRoute();
const router = useRouter();
const { showToast } = useToastService();

const action = route.params.action as string;
const reportId = route.params.reportId as string;
const config = getExpenseAction(action);
const submitting = ref(false);

// Loading this page must never change state; the POST only fires from an explicit click.
onMounted(async () => {
  if (!config) {
    showToast(`Unknown action "${action}". Please use the link from your email.`, ToastTypesEnum.negative);
    await router.replace('/');
  }
});

const confirm = async () => {
  if (!config || submitting.value) return;
  submitting.value = true;

  try {
    await $fetch(`/api/expense-email/${encodeURIComponent(action)}/${encodeURIComponent(reportId)}`, {
      method: 'POST',
    });
    showToast(`The expense report has been ${config.past}.`, ToastTypesEnum.positive);
  } catch {
    showToast(`Failed to ${config.verb} the expense report. Please contact LF Support.`, ToastTypesEnum.negative);
  }

  await router.replace('/');
};
</script>
