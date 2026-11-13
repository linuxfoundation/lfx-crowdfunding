// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { defineNuxtPlugin, useAsyncData, useRoute, navigateTo, onNuxtReady } from 'nuxt/app';
import { watch } from 'vue';
import {
  authState,
  isAuthLoading,
  isAuthReady,
  setProfileSynced,
  setRefreshAuth,
} from '~/composables/useAuth';
import { useErrorToast } from '~/composables/useErrorToast';
import { profileSyncErrorMessage } from '~/utils/profile-sync-error';
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
        // Mirror userData → authState synchronously before raising isAuthReady,
        // so any code that awaits isAuthReady already sees the correct authState.
        // This removes the need for a nextTick() workaround in handleAuthQuery.
        if (userData.value) {
          authState.value = userData.value;
        }
        isAuthLoading.value = false;
        isAuthReady.value = true; // set last — watchers on this see consistent authState
      }
    },
    { immediate: true },
  );

  const { showError } = useErrorToast();

  const route = useRoute();

  // authState is guaranteed to be populated once this resolves: watch(status) sets it
  // synchronously before raising isAuthReady (see above).
  const waitForAuthReady = async () => {
    if (isAuthReady.value) return;
    await new Promise<void>((resolve) => {
      const stop = watch(isAuthReady, (ready) => {
        if (!ready) return;
        stop();
        resolve();
      });
    });
  };

  const syncProfile = async () => {
    // wait for auth state to be hydrated before syncing the profile so the
    // session cookie is guaranteed to be present server-side.
    await waitForAuthReady();

    if (authState.value.isAuthenticated) {
      await $fetch('/api/me', { method: 'PATCH', credentials: 'include' }).catch((err) => {
        console.error('Profile sync error:', err);
        showError(profileSyncErrorMessage(err));
      });
    }
  };

  // Strip ?auth=... from the URL without triggering a navigation.
  const stripAuthQuery = async () => {
    const { auth: _auth, ...rest } = route.query;
    await navigateTo({ path: route.path, query: rest }, { replace: true });
  };

  const handleAuthQuery = async (authParam: string | undefined) => {
    if (authParam === 'logout') {
      await navigateTo('/', { replace: true });
      return;
    }

    if (authParam === 'success') {
      // Registered synchronously (before any await) so pages mounting after login can
      // wait for the users row to exist before calling endpoints that read it.
      const sync = syncProfile();
      setProfileSynced(sync);
      await sync;
      await stripAuthQuery();
      return;
    }

    if (authParam === 'error') {
      // The login callback was rejected (stale or tampered login, or an Auth0 error). A user
      // who still has a session, e.g. after pressing Back onto the callback, needs no message.
      await waitForAuthReady();
      if (!authState.value.isAuthenticated) {
        // The toast component (in the layout) drops messages sent before it has mounted.
        onNuxtReady(() => showError('We could not log you in. Please try logging in again.'));
      }
      await stripAuthQuery();
    }
  };

  const runAuthQuery = (authParam: string | undefined) => {
    handleAuthQuery(authParam).catch((err) => console.error('Auth query handling error:', err));
  };

  if (['logout', 'success', 'error'].includes(route.query.auth as string)) {
    runAuthQuery(route.query.auth as string);
  }

  watch(
    () => route.query.auth,
    (authParam) => runAuthQuery(authParam as string | undefined),
  );
});
