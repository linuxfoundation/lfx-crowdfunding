// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { watch } from 'vue';
import { isAuthReady, authState, login } from '~/composables/useAuth';

export default defineNuxtRouteMiddleware(async (to) => {
  if (!import.meta.client) return;

  if (!isAuthReady.value) {
    await new Promise<void>((resolve) => {
      const stop = watch(isAuthReady, (ready) => {
        if (!ready) return;
        stop();
        resolve();
      });
    });
  }

  if (!authState.value.isAuthenticated) {
    await login(to.fullPath);
    return abortNavigation();
  }
});
