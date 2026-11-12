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
  // Duck-typed DOM nodes: the suite runs in the node environment.
  const node = (nodeName: string, attrs: Record<string, string> = {}) => ({
    nodeName,
    attributes: Object.entries(attrs).map(([name, value]) => ({ name, value })),
  });
  // Shape of Vue's prod console.warn for an attribute mismatch (string, element, string).
  const img = node('IMG', { src: '/_nuxt/logo.svg', alt: 'Jenkins' });
  const detail =
    '\n  - rendered on server: src="/_nuxt/logo.svg"\n  - expected on client: src="https://x/_nuxt/logo.svg"';

  it('flattens a hydration warning, keeping the src mismatch and redacting other attributes', () => {
    const message = hydrationWarningMessage([
      '[Vue warn]: Hydration attribute mismatch on',
      img,
      detail,
    ]);
    expect(message).toContain(
      '[Vue warn]: Hydration attribute mismatch on <img src="/_nuxt/logo.svg" alt="[redacted]">',
    );
    expect(message).toContain('rendered on server: src="/_nuxt/logo.svg"');
    expect(message).toContain('expected on client: src="https://x/_nuxt/logo.svg"');
    expect(message).not.toContain('Jenkins');
  });

  it('redacts the server and client text of a text-content mismatch', () => {
    const message = hydrationWarningMessage([
      '[Vue warn]: Hydration text content mismatch on',
      node('SPAN', { class: 'donor-name' }),
      '\n  - rendered on server: Jane Doe <jane.doe@example.com>\n  - expected on client: Jane',
    ]);
    expect(message).toBe(
      '[Vue warn]: Hydration text content mismatch on <span class="donor-name"> \n' +
        '  - rendered on server: [redacted]\n  - expected on client: [redacted]',
    );
  });

  it('redacts multi-line text from a text-node mismatch', () => {
    const message = hydrationWarningMessage([
      '[Vue warn]: Hydration text mismatch in',
      node('P'),
      '\n  - rendered on server: "Thanks,\\nJane Doe"\n  - expected on client: "Thanks,\nJane Doe\njane@example.com"',
    ]);
    expect(message).not.toMatch(/Jane|jane|Thanks/);
    expect(message).toContain('- rendered on server: [redacted]');
  });

  it('redacts the value of a populated input', () => {
    const message = hydrationWarningMessage([
      '[Vue warn]: Hydration attribute mismatch on',
      node('INPUT', { type: 'email', value: 'jane.doe@example.com' }),
      '\n  - rendered on server: value="jane.doe@example.com"\n  - expected on client: (not rendered)',
    ]);
    expect(message).not.toContain('jane.doe@example.com');
    expect(message).toContain('<input type="email" value="[redacted]">');
    expect(message).toContain('- rendered on server: value="[redacted]"');
    expect(message).toContain('- expected on client: (not rendered)');
  });

  it("keeps a node mismatch's tag names and Vue's fixed wording", () => {
    const message = hydrationWarningMessage([
      '[Vue warn]: Hydration node mismatch:\n- rendered on server:',
      { nodeName: '#text' },
      '(text)',
      '\n- expected on client:',
      'div',
    ]);
    expect(message).toBe(
      '[Vue warn]: Hydration node mismatch:\n- rendered on server: #text (text) \n- expected on client: div',
    );
  });

  it('names a component vnode type without its contents', () => {
    const message = hydrationWarningMessage([
      '[Vue warn]: Hydration node mismatch:\n- rendered on server:',
      node('DIV'),
      '',
      '\n- expected on client:',
      { __name: 'DonorList', props: { secret: 'jane@example.com' } },
    ]);
    expect(message).toContain('- expected on client: <DonorList>');
    expect(message).not.toContain('jane');
  });

  it('ignores other warnings', () => {
    expect(hydrationWarningMessage(['[Vue warn]: Extraneous non-props attributes'])).toBeNull();
    expect(hydrationWarningMessage([img])).toBeNull();
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
