// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { DatadogIdentifiedUser } from '~/types/datadog.types';

// Thin wrapper around the Datadog RUM SDK exposing user-identity helpers.
// All methods are SSR-safe — they are no-ops on the server or when RUM is not
// initialised (e.g. local dev without DD credentials).

async function getRum() {
  if (typeof window === 'undefined') return null;
  const { datadogRum } = await import('@datadog/browser-rum');
  return datadogRum;
}

export const useDatadogRum = () => {
  async function setUser(user: DatadogIdentifiedUser): Promise<void> {
    const rum = await getRum();
    rum?.setUser(user);
  }

  async function clearUser(): Promise<void> {
    const rum = await getRum();
    rum?.clearUser();
  }

  return { setUser, clearUser };
};
