// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Thin OpenFeature/LaunchDarkly wrapper. SSR-safe — everything here is a
// no-op on the server or before the client plugin has initialized.
import { ref, computed, type ComputedRef } from 'vue';
import {
  OpenFeature,
  ProviderEvents,
  type Client,
  type EvaluationContext,
} from '@openfeature/web-sdk';

let client: Client | null = null;
const isReady = ref(false);
// Bumped on every provider event so computed flag refs re-evaluate.
const revision = ref(0);

export interface FeatureFlagUser {
  username?: string;
  name?: string;
  email?: string;
}

function toContext(user?: FeatureFlagUser): EvaluationContext {
  return {
    kind: 'user',
    targetingKey: user?.username || 'anonymous',
    anonymous: !user?.username,
    name: user?.name,
    email: user?.email,
  };
}

export async function initFeatureFlags(clientId: string): Promise<void> {
  if (!clientId || client) return;

  const { LaunchDarklyClientProvider } = await import('@openfeature/launchdarkly-client-provider');
  const provider = new LaunchDarklyClientProvider(clientId, { streaming: true });

  await OpenFeature.setContext(toContext());
  await OpenFeature.setProviderAndWait(provider);

  client = OpenFeature.getClient();
  isReady.value = true;
  revision.value++;

  client.addHandler(ProviderEvents.ConfigurationChanged, () => revision.value++);
  client.addHandler(ProviderEvents.ContextChanged, () => revision.value++);
  client.addHandler(ProviderEvents.Error, () => console.error('[FeatureFlags] Provider error'));
}

// Call once the authenticated user is known, to move off the anonymous context.
export async function identifyFeatureFlagUser(user: FeatureFlagUser): Promise<void> {
  if (!client) return;
  await OpenFeature.setContext(toContext(user));
  revision.value++;
}

export const useFeatureFlags = () => {
  function getBooleanFlag(key: string, defaultValue = false): ComputedRef<boolean> {
    return computed(() => {
      void revision.value; // reactive dependency
      return client?.getBooleanValue(key, defaultValue) ?? defaultValue;
    });
  }

  return { ready: isReady, getBooleanFlag };
};
