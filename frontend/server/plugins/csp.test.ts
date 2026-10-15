// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { describe, it, expect, vi, beforeEach } from 'vitest';
import type { H3Event } from 'h3';

const setResponseHeader = vi.fn();
const publicConfig: Record<string, string | undefined> = {};

vi.mock('h3', () => ({ setResponseHeader }));
vi.mock('nitropack/runtime', () => ({
  defineNitroPlugin: (fn: unknown) => fn,
  useRuntimeConfig: () => ({ public: publicConfig }),
}));

const { setCspHeader } = await import('./csp');

const event = (cspNonce?: string) => ({ context: { cspNonce } }) as unknown as H3Event;

describe('setCspHeader', () => {
  beforeEach(() => {
    setResponseHeader.mockClear();
    for (const k of Object.keys(publicConfig)) delete publicConfig[k];
  });

  it('sends the report-only policy with the nonce from the request', () => {
    setCspHeader(event('abc'));
    expect(setResponseHeader.mock.lastCall?.[1]).toBe('Content-Security-Policy-Report-Only');
    expect(setResponseHeader.mock.lastCall?.[2]).toContain("script-src 'nonce-abc'");
  });

  it('sends nothing when the app set no nonce', () => {
    setCspHeader(event());
    expect(setResponseHeader).not.toHaveBeenCalled();
  });

  it('sends report-uri only when the RUM client token is set', () => {
    setCspHeader(event('abc'));
    expect(setResponseHeader.mock.lastCall?.[2]).not.toContain('report-uri');
    Object.assign(publicConfig, { datadogRumClientToken: 'tok', appEnv: 'staging' });
    setCspHeader(event('abc'));
    expect(setResponseHeader.mock.lastCall?.[2]).toContain(
      'report-uri https://browser-intake-datadoghq.com',
    );
  });
});
