// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Thin OpenFeature/LaunchDarkly wrapper. SSR-safe — everything here is a
// no-op on the server or before the client plugin has initialized.
import { ref, computed, type ComputedRef } from 'vue';
import {
  OpenFeature,
  ProviderEvents,
  ProviderStatus,
  type Client,
  type EvaluationContext,
} from '@openfeature/web-sdk';

const ANONYMOUS_KEY_STORAGE_KEY = 'lfx-ff-anon-id';

let client: Client | null = null;
const isReady = ref(false);
// Bumped on every provider event so computed flag refs re-evaluate.
const revision = ref(0);
// Last user requested via identifyFeatureFlagUser before the client was ready; replayed once init finishes.
let pendingUser: FeatureFlagUser | undefined;

export interface FeatureFlagUser {
  username?: string;
  name?: string;
  email?: string;
}

// Persist one anonymous key per browser so anonymous visitors are distributed across
// LaunchDarkly percentage rollouts instead of all hashing to the same targeting key.
function getAnonymousKey(): string {
  if (typeof window === 'undefined') return 'anonymous';

  const existing = window.localStorage.getItem(ANONYMOUS_KEY_STORAGE_KEY);
  if (existing) return existing;

  const generated = crypto.randomUUID();
  window.localStorage.setItem(ANONYMOUS_KEY_STORAGE_KEY, generated);
  return generated;
}

function toContext(user?: FeatureFlagUser): EvaluationContext {
  return {
    kind: 'user',
    targetingKey: user?.username || getAnonymousKey(),
    anonymous: !user?.username,
    name: user?.name,
    email: user?.email,
  };
}

export async function initFeatureFlags(clientId: string): Promise<void> {
  if (!clientId || client) return;

  const { LaunchDarklyClientProvider } = await import('@openfeature/launchdarkly-client-provider');
  const provider = new LaunchDarklyClientProvider(clientId, { streaming: true });

  await OpenFeature.setContext(toContext(pendingUser));
  await OpenFeature.setProviderAndWait(provider);

  // setProviderAndWait resolves even when the provider failed to initialize (it swallows
  // the error and sets its own status to ERROR), so check status explicitly.
  if (OpenFeature.getProviderStatus() !== ProviderStatus.READY) {
    throw new Error(
      `LaunchDarkly provider failed to initialize (status: ${OpenFeature.getProviderStatus()})`,
    );
  }

  client = OpenFeature.getClient();
  isReady.value = true;
  revision.value++;

  // Re-apply in case identifyFeatureFlagUser was called while setProviderAndWait was in flight.
  if (pendingUser) {
    await OpenFeature.setContext(toContext(pendingUser));
    revision.value++;
  }

  client.addHandler(ProviderEvents.ConfigurationChanged, () => revision.value++);
  client.addHandler(ProviderEvents.ContextChanged, () => revision.value++);
  client.addHandler(ProviderEvents.Error, (details) =>
    console.error('[FeatureFlags] Provider error', details),
  );
}

// Call once the authenticated user is known, to move off the anonymous context.
export async function identifyFeatureFlagUser(user: FeatureFlagUser): Promise<void> {
  pendingUser = user;
  if (!client) return;
  await OpenFeature.setContext(toContext(user));
  revision.value++;
}

// Call on logout to drop back to the anonymous context, so a previously identified
// user's targeting doesn't leak into the rest of the (now unauthenticated) session.
export async function resetFeatureFlagUser(): Promise<void> {
  pendingUser = undefined;
  if (!client) return;
  await OpenFeature.setContext(toContext());
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
