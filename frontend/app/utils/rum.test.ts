// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { describe, expect, it } from 'vitest';
import type { RumEvent } from '@datadog/browser-rum';
import { hydrationWarningMessage, isInjectedEvalCspReport, isTranslatedPage } from './rum';

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

describe('hydrationWarningMessage', () => {
  // Shape of Vue's prod console.warn for an attribute mismatch (string, element, string).
  const element = { outerHTML: '<img src="/_nuxt/logo.svg" alt="Jenkins"><span>child</span>' };
  const detail =
    '\n  - rendered on server: src="/_nuxt/logo.svg"\n  - expected on client: src="https://x/_nuxt/logo.svg"';

  it('flattens a hydration warning, keeping only the element opening tag', () => {
    const message = hydrationWarningMessage([
      '[Vue warn]: Hydration attribute mismatch on',
      element,
      detail,
    ]);
    expect(message).toContain(
      '[Vue warn]: Hydration attribute mismatch on <img src="/_nuxt/logo.svg" alt="Jenkins">',
    );
    expect(message).toContain('expected on client: src="https://x/_nuxt/logo.svg"');
    expect(message).not.toContain('child');
  });

  it('ignores other warnings', () => {
    expect(hydrationWarningMessage(['[Vue warn]: Extraneous non-props attributes'])).toBeNull();
    expect(hydrationWarningMessage([element])).toBeNull();
    expect(hydrationWarningMessage([])).toBeNull();
  });
});

describe('isTranslatedPage', () => {
  const root = (lang: string | null, classes: string[] = []) => ({
    classList: { contains: (name: string) => classes.includes(name) } as DOMTokenList,
    getAttribute: () => lang,
  });

  it('detects the Chrome Translate class', () => {
    expect(isTranslatedPage(root('en', ['translated-ltr']))).toBe(true);
    expect(isTranslatedPage(root('en', ['translated-rtl']))).toBe(true);
  });

  it('detects a non-English lang', () => {
    expect(isTranslatedPage(root('ms'))).toBe(true);
    expect(isTranslatedPage(root('ar'))).toBe(true);
  });

  it('treats English as untranslated', () => {
    expect(isTranslatedPage(root('en'))).toBe(false);
    expect(isTranslatedPage(root('en-US'))).toBe(false);
    expect(isTranslatedPage(root(null))).toBe(false);
  });
});
