// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { describe, expect, it } from 'vitest';
import type { RumEvent } from '@datadog/browser-rum';
import { isInjectedEvalCspReport } from './rum';

const EVAL = "csp_violation: 'eval' blocked by 'script-src' directive";

const errorEvent = (error: Record<string, unknown>) =>
  ({ type: 'error', error: { source: 'report', message: EVAL, ...error } }) as unknown as RumEvent;

describe('isInjectedEvalCspReport', () => {
  it('matches an eval report with no source file', () => {
    expect(isInjectedEvalCspReport(errorEvent({}))).toBe(true);
  });

  it('keeps an eval report from one of our scripts', () => {
    const stack = 'script-src: ...\n  at https://crowdfunding.linuxfoundation.org/_nuxt/a.js:1:1';
    expect(isInjectedEvalCspReport(errorEvent({ stack }))).toBe(false);
  });

  it('keeps other CSP violations', () => {
    const message = "csp_violation: 'https://evil.example' blocked by 'connect-src' directive";
    expect(isInjectedEvalCspReport(errorEvent({ message }))).toBe(false);
  });

  it('keeps errors that are not reports', () => {
    expect(isInjectedEvalCspReport(errorEvent({ source: 'source' }))).toBe(false);
  });
});
