// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { describe, it, expect, vi } from 'vitest';

vi.stubGlobal('defineNuxtPlugin', (fn: unknown) => fn);
const { default: plugin } = (await import('./csp-nonce.server')) as unknown as {
  default: (nuxtApp: unknown) => void;
};

describe('csp-nonce plugin', () => {
  it('stores a fresh nonce and sets it on script tags only', () => {
    let resolve: (ctx: {
      tags: { tag: string; props: Record<string, string> }[];
    }) => void = () => {};
    const context: Record<string, string> = {};
    plugin({
      ssrContext: {
        event: { context },
        head: { hooks: { hook: (_: string, fn: typeof resolve) => (resolve = fn) } },
      },
    });

    const tags = [
      { tag: 'script', props: { src: '/_nuxt/entry.js' } },
      { tag: 'meta', props: { content: 'x <script y=><img src=x onerror=alert(1)>' } },
    ];
    resolve({ tags });

    expect(context.cspNonce).toMatch(/^[A-Za-z0-9+/]{22}==$/);
    expect(tags[0]!.props.nonce).toBe(context.cspNonce);
    expect(tags[1]!.props).not.toHaveProperty('nonce');
  });

  it('does nothing on the client', () => {
    expect(() => plugin({})).not.toThrow();
  });
});
