// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { defineNuxtPlugin, useAsyncData, useRoute, navigateTo } from 'nuxt/app';
import { watch, watchEffect, nextTick } from 'vue';
import { authState, isAuthLoading, isAuthReady, setRefreshAuth } from '~/composables/useAuth';
import type { AuthState } from '~/composables/useAuth';

export default defineNuxtPlugin(() => {
  const {
    data: userData,
    refresh: refreshAuth,
    status,
  } = useAsyncData<AuthState>(
    'auth-user',
    () => $fetch('/api/auth/user', { credentials: 'include' }),
    {
      default: () => ({ isAuthenticated: false, user: null, token: null }),
      server: false,
      lazy: true,
    },
  );

  setRefreshAuth(refreshAuth);

  watch(
    status,
    (s) => {
      if (s === 'success' || s === 'error') {
        isAuthReady.value = true;
        isAuthLoading.value = false;
      }
    },
    { immediate: true },
  );

  watchEffect(() => {
    if (!userData.value) return;
    isAuthLoading.value = false;
    authState.value = userData.value;
  });

  const route = useRoute();

  const handleAuthQuery = async (authParam: string | undefined) => {
    if (authParam === 'logout') {
      await navigateTo('/', { replace: true });
      return;
    }

    if (authParam === 'success') {
      // Wait for auth state to be hydrated before syncing the profile so the
      // session cookie is guaranteed to be present server-side.
      if (!isAuthReady.value) {
        await new Promise<void>((resolve) => {
          const stop = watch(isAuthReady, (ready) => {
            if (!ready) return;
            stop();
            resolve();
          });
        });
      }

      if (authState.value.isAuthenticated) {
        await $fetch('/api/me', { method: 'PATCH', credentials: 'include' }).catch((err) =>
          console.error('Profile sync error:', err),
        );
      }

      // Strip ?auth=success from the URL without triggering a navigation.
      const { auth: _auth, ...rest } = route.query;
      await navigateTo({ path: route.path, query: rest }, { replace: true });
    }
  };

  const runAuthQuery = (authParam: string | undefined) => {
    handleAuthQuery(authParam).catch((err) => console.error('Auth query handling error:', err));
  };

  if (route.query.auth === 'logout' || route.query.auth === 'success') {
    nextTick(() => runAuthQuery(route.query.auth as string | undefined));
  }

  watch(
    () => route.query.auth,
    (authParam) => runAuthQuery(authParam as string | undefined),
  );
});
