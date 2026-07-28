// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { vi, describe, it, expect, beforeEach } from 'vitest';

// --- Mock the OpenFeature SDK and the (dynamically imported) LaunchDarkly provider ---
// useFeatureFlags is a module-level singleton (client/isReady/pendingUser live outside
// the composable), so each test reloads the module fresh via vi.resetModules() below.

const mockSetContext = vi.fn();
const mockSetProviderAndWait = vi.fn();
const mockGetBooleanValue = vi.fn();
const handlers: Record<string, Array<(details?: unknown) => void>> = {};

// The wrapper checks the provider *instance's* own `status`, not OpenFeature's wrapper
// status (see useFeatureFlags.ts) — so the fake provider instance exposes its own status.
let mockProviderStatus = 'READY';

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
  },
  ProviderEvents: {
    ConfigurationChanged: 'configuration_changed',
    ContextChanged: 'context_changed',
    Error: 'error',
  },
  ProviderStatus: { READY: 'READY', ERROR: 'ERROR' },
}));

vi.mock('@openfeature/launchdarkly-client-provider', () => ({
  LaunchDarklyClientProvider: vi.fn().mockImplementation(function (this: { status: string }) {
    this.status = mockProviderStatus;
  }),
}));

async function loadModule() {
  return import('./useFeatureFlags');
}

describe('useFeatureFlags', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.resetModules();
    for (const key of Object.keys(handlers)) delete handlers[key];
    mockProviderStatus = 'READY';
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

  it('applies a reset requested while init is still in flight', async () => {
    const { initFeatureFlags, identifyFeatureFlagUser, resetFeatureFlagUser } = await loadModule();

    await identifyFeatureFlagUser({ username: 'alice' });

    let resolveWait!: () => void;
    mockSetProviderAndWait.mockReturnValueOnce(
      new Promise<void>((resolve) => {
        resolveWait = resolve;
      }),
    );

    const initPromise = initFeatureFlags('client-id');
    await resetFeatureFlagUser(); // logout happens before the provider finishes initializing
    resolveWait();
    await initPromise;

    const contexts = mockSetContext.mock.calls.map((call) => call[0]);
    expect(contexts.at(-1)).toMatchObject({ anonymous: true });
  });

  it('falls back to an in-memory anonymous key when localStorage throws', async () => {
    // @ts-expect-error — minimal window stub for a sandboxed/storage-blocked origin
    globalThis.window = {
      localStorage: {
        getItem: () => {
          throw new DOMException('blocked', 'SecurityError');
        },
        setItem: () => {
          throw new DOMException('blocked', 'SecurityError');
        },
      },
    };

    try {
      const { initFeatureFlags } = await loadModule();
      await expect(initFeatureFlags('client-id')).resolves.toBeUndefined();

      const context = mockSetContext.mock.calls[0]?.[0];
      expect(context).toMatchObject({ anonymous: true });
      expect(typeof context.targetingKey).toBe('string');
    } finally {
      // @ts-expect-error — restore to the default 'window is undefined' node environment
      delete globalThis.window;
    }
  });

  it('throws and stays unready when the provider fails to initialize', async () => {
    mockProviderStatus = 'ERROR';
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
