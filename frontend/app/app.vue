<!--
Copyright (c) 2025 The Linux Foundation and each contributor.
SPDX-License-Identifier: MIT
-->
<template>
  <NuxtLayout>
    <NuxtPage />
  </NuxtLayout>
</template>

<script setup lang="ts">
import { onMounted, watch } from 'vue';
import { authState, isAuthReady } from '~/composables/useAuth';
import { useDatadogRum } from '~/composables/useDatadogRum';
import { identifyFeatureFlagUser } from '~/composables/useFeatureFlags';
import { useIntercom } from '~/composables/useIntercom';

const { boot, shutdown } = useIntercom();
const { setUser: setDdUser, clearUser: clearDdUser } = useDatadogRum();

// Guard against re-booting with identity on every auth state change.
let intercomBootAttempted = false;

function bootAnonymous() {
  boot({}).catch((err) => {
    console.warn('[App] Anonymous Intercom boot failed', err);
  });
}

onMounted(() => {
  // Boot anonymously so banners/popups are visible to all visitors before login.
  bootAnonymous();
});

// Upgrade to identified session once auth is known, or shutdown on logout.
watch(
  [isAuthReady, () => authState.value.isAuthenticated, () => authState.value.user],
  ([ready, isAuthenticated, user]) => {
    if (!ready) return;

    if (isAuthenticated && user && !intercomBootAttempted) {
      const { username, intercomJwt, name, email } = user;

      // Identify the user in Datadog RUM and LaunchDarkly.
      if (username) {
        setDdUser({ id: username, email, name });
        identifyFeatureFlagUser({ username, name, email }).catch((err) => {
          console.error('[App] Failed to identify feature flag user', err);
        });
      }

      if (username && intercomJwt) {
        intercomBootAttempted = true;
        boot({
          user_id: username,
          intercom_user_jwt: intercomJwt,
          name,
          email,
        }).catch((err) => {
          console.error('[App] Identified Intercom boot failed', err);
          intercomBootAttempted = false;
        });
      } else {
        console.warn('[App] Intercom not booted — missing username or intercomJwt claim', {
          hasUsername: !!username,
          hasIntercomJwt: !!intercomJwt,
        });
      }
    } else if (!isAuthenticated) {
      clearDdUser();
      if (intercomBootAttempted) {
        shutdown();
        intercomBootAttempted = false;
        bootAnonymous();
      }
    }
  },
  { immediate: true },
);
</script>
