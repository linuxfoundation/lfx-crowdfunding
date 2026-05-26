<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <div class="flex items-center justify-center min-h-[60vh] px-4">
    <!-- Auth loading -->
    <template v-if="!isAuthReady">
      <lfx-spinner :size="32" />
    </template>

    <!-- Initiative loading -->
    <template v-else-if="initiativePending">
      <lfx-spinner :size="32" />
    </template>

    <!-- Invalid action -->
    <template v-else-if="!isValidAction">
      <lfx-card class="max-w-md w-full text-center p-8 flex flex-col gap-4">
        <lfx-icon
          name="triangle-exclamation"
          type="regular"
          :size="40"
          class="text-yellow-500 mx-auto"
        />
        <h1 class="text-lg font-semibold text-neutral-900">Invalid approval link</h1>
        <p class="text-sm text-neutral-500">This link is not valid. Please check the email and try again.</p>
      </lfx-card>
    </template>

    <!-- Initiative fetch error -->
    <template v-else-if="initiativeError">
      <lfx-card class="max-w-md w-full text-center p-8 flex flex-col gap-4">
        <lfx-icon
          name="circle-exclamation"
          type="regular"
          :size="40"
          class="text-red-500 mx-auto"
        />
        <h1 class="text-lg font-semibold text-neutral-900">Initiative not found</h1>
        <p class="text-sm text-neutral-500">
          We couldn't find this initiative. It may have been removed or the link may be incorrect.
        </p>
      </lfx-card>
    </template>

    <!-- Success -->
    <template v-else-if="result">
      <lfx-card class="max-w-md w-full text-center p-8 flex flex-col gap-4">
        <lfx-icon
          :name="action === 'approve' ? 'circle-check' : 'circle-xmark'"
          type="regular"
          :size="40"
          :class="action === 'approve' ? 'text-green-500' : 'text-red-500'"
          class="mx-auto"
        />
        <h1 class="text-lg font-semibold text-neutral-900">
          Initiative {{ action === 'approve' ? 'approved' : 'declined' }}
        </h1>
        <p class="text-sm text-neutral-500">
          <strong>{{ result.name }}</strong> has been
          {{ action === 'approve' ? 'published and is now live' : 'declined' }}. The initiative owner has been notified
          by email.
        </p>
        <NuxtLink
          v-if="action === 'approve'"
          :to="`/initiatives/${result.slug}`"
        >
          <lfx-button
            label="View initiative"
            type="primary"
            class="mx-auto"
          />
        </NuxtLink>
      </lfx-card>
    </template>

    <!-- API error -->
    <template v-else-if="apiError">
      <lfx-card class="max-w-md w-full text-center p-8 flex flex-col gap-4">
        <lfx-icon
          name="circle-exclamation"
          type="regular"
          :size="40"
          class="text-red-500 mx-auto"
        />
        <h1 class="text-lg font-semibold text-neutral-900">Something went wrong</h1>
        <p class="text-sm text-neutral-500">{{ apiError }}</p>
        <lfx-button
          label="Try again"
          type="secondary"
          class="mx-auto"
          @click="submit"
        />
      </lfx-card>
    </template>

    <!-- Confirmation -->
    <template v-else-if="initiative">
      <lfx-card class="max-w-md w-full p-8 flex flex-col gap-6">
        <div class="flex flex-col gap-2 text-center">
          <lfx-icon
            :name="action === 'approve' ? 'circle-check' : 'circle-xmark'"
            type="light"
            :size="40"
            :class="action === 'approve' ? 'text-green-500' : 'text-red-500'"
            class="mx-auto"
          />
          <h1 class="text-lg font-semibold text-neutral-900">
            {{ action === 'approve' ? 'Approve' : 'Decline' }} initiative
          </h1>
        </div>

        <div class="flex flex-col gap-1">
          <p class="text-sm text-neutral-600">
            You are about to
            <strong>{{ action === 'approve' ? 'approve' : 'decline' }}</strong>
            the following initiative:
          </p>
          <p class="text-sm font-semibold text-neutral-900">{{ initiative.name }}</p>
        </div>

        <p class="text-xs text-neutral-400">
          <template v-if="action === 'approve'">
            Approving will publish this initiative and make it publicly visible. The owner will receive a confirmation
            email.
          </template>
          <template v-else>
            Declining will prevent this initiative from being published. The owner will receive a rejection email.
          </template>
        </p>

        <div class="flex gap-3 justify-center">
          <lfx-button
            :label="action === 'approve' ? 'Approve' : 'Decline'"
            :type="action === 'approve' ? 'primary' : 'secondary'"
            :loading="processing"
            @click="submit"
          />
        </div>
      </lfx-card>
    </template>
  </div>
</template>

<script setup lang="ts">
import LfxButton from '~/components/uikit/button/button.vue';
import LfxCard from '~/components/uikit/card/card.vue';
import LfxIcon from '~/components/uikit/icon/icon.vue';
import LfxSpinner from '~/components/uikit/spinner/spinner.vue';
import { useAuth, isAuthReady } from '~/composables/useAuth';
import type { ApprovalAction, ApprovalResult } from '~/types/approval.types';
import type { InitiativeDetail } from '#shared/types/initiative-detail.types';

const props = defineProps<{
  slug: string;
  action: string;
}>();

const isValidAction = computed(() => props.action === 'approve' || props.action === 'decline');
const action = computed(() => props.action as ApprovalAction);

const { isAuthenticated, login } = useAuth();

watch(
  isAuthReady,
  (ready) => {
    if (ready && !isAuthenticated.value) {
      login();
    }
  },
  { immediate: true },
);

const {
  data: initiative,
  pending: initiativePending,
  error: initiativeError,
} = useFetch<InitiativeDetail>(`/api/initiatives/${props.slug}`, {
  immediate: isValidAction.value,
});

const processing = ref(false);
const result = ref<ApprovalResult | null>(null);
const apiError = ref<string | null>(null);

async function submit() {
  processing.value = true;
  apiError.value = null;
  try {
    result.value = await $fetch<ApprovalResult>(`/api/initiatives/${props.slug}/process-approval/${props.action}`, {
      method: 'POST',
    });
  } catch (err: unknown) {
    const e = err as { data?: { message?: string }; statusMessage?: string };
    apiError.value = e?.data?.message ?? e?.statusMessage ?? 'An unexpected error occurred. Please try again.';
  } finally {
    processing.value = false;
  }
}
</script>
