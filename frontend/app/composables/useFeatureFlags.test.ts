// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { vi, describe, it, expect, beforeEach } from 'vitest';

// --- Mock the OpenFeature SDK and the (dynamically imported) LaunchDarkly provider ---
// useFeatureFlags is a module-level singleton (client/isReady/pendingUser live outside
// the composable), so each test reloads the module fresh via vi.resetModules() below.

const mockSetContext = vi.fn();
const mockSetProviderAndWait = vi.fn();
const mockGetProviderStatus = vi.fn();
const mockGetBooleanValue = vi.fn();
const handlers: Record<string, Array<(details?: unknown) => void>> = {};

const mockClient = {
  getBooleanValue: (...args: unknown[]) => mockGetBooleanValue(...args),
  addHandler: (event: string, cb: (details?: unknown) => void) => {
    (handlers[event] ??= []).push(cb);
  },
};

vi.mock('@openfeature/web-sdk', () => ({
  OpenFeature: {
    setContext: (...args: unknown[]) => mockSetContext(...args),
    setProviderAndWait: (...args: unknown[]) => mockSetProviderAndWait(...args),
    getClient: () => mockClient,
    getProviderStatus: () => mockGetProviderStatus(),
  },
  ProviderEvents: {
    ConfigurationChanged: 'configuration_changed',
    ContextChanged: 'context_changed',
    Error: 'error',
  },
  ProviderStatus: { READY: 'READY', ERROR: 'ERROR' },
}));

vi.mock('@openfeature/launchdarkly-client-provider', () => ({
  LaunchDarklyClientProvider: vi.fn(),
}));

async function loadModule() {
  return import('./useFeatureFlags');
}

describe('useFeatureFlags', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.resetModules();
    for (const key of Object.keys(handlers)) delete handlers[key];
    mockGetProviderStatus.mockReturnValue('READY');
    mockSetContext.mockResolvedValue(undefined);
    mockSetProviderAndWait.mockResolvedValue(undefined);
  });

  it('returns the default value before the provider is ready', async () => {
    const { useFeatureFlags } = await loadModule();

    const flag = useFeatureFlags().getBooleanFlag('my-flag', true);

    expect(flag.value).toBe(true);
    expect(mockGetBooleanValue).not.toHaveBeenCalled();
  });

  it('buffers an identify call made before init and replays it once the provider is ready', async () => {
    const { initFeatureFlags, identifyFeatureFlagUser } = await loadModule();

    await identifyFeatureFlagUser({ username: 'alice', name: 'Alice', email: 'alice@example.com' });
    await initFeatureFlags('client-id');

    const contexts = mockSetContext.mock.calls.map((call) => call[0]);
    expect(contexts.at(-1)).toMatchObject({ targetingKey: 'alice', anonymous: false });
  });

  it('resets to the anonymous context on logout', async () => {
    const { initFeatureFlags, identifyFeatureFlagUser, resetFeatureFlagUser } = await loadModule();

    await initFeatureFlags('client-id');
    await identifyFeatureFlagUser({ username: 'alice' });
    mockSetContext.mockClear();

    await resetFeatureFlagUser();

    expect(mockSetContext).toHaveBeenCalledWith(expect.objectContaining({ anonymous: true }));
  });

  it('throws and stays unready when the provider fails to initialize', async () => {
    mockGetProviderStatus.mockReturnValue('ERROR');
    const { initFeatureFlags, useFeatureFlags } = await loadModule();

    await expect(initFeatureFlags('client-id')).rejects.toThrow(/failed to initialize/i);
    expect(useFeatureFlags().ready.value).toBe(false);

    // getBooleanFlag still falls back to the default since client was never set.
    const flag = useFeatureFlags().getBooleanFlag('my-flag', false);
    expect(flag.value).toBe(false);
    expect(mockGetBooleanValue).not.toHaveBeenCalled();
  });

  it('re-evaluates flags reactively when the provider emits an event', async () => {
    const { initFeatureFlags, useFeatureFlags } = await loadModule();
    await initFeatureFlags('client-id');

    mockGetBooleanValue.mockReturnValue(false);
    const flag = useFeatureFlags().getBooleanFlag('my-flag', false);
    expect(flag.value).toBe(false);

    mockGetBooleanValue.mockReturnValue(true);
    handlers['configuration_changed']?.forEach((cb) => cb());

    expect(flag.value).toBe(true);
  });
});
