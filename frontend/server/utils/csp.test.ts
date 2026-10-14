// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { describe, it, expect } from 'vitest';
import { addScriptNonce, buildCspPolicy, datadogCspReportUri } from './csp';

describe('addScriptNonce', () => {
  it('stamps every script tag, inline and external', () => {
    const out = addScriptNonce('<script>a()</script><script src="/x.js" async></script>', 'N');
    expect(out).toBe('<script nonce="N">a()</script><script nonce="N" src="/x.js" async></script>');
  });

  it('leaves tags that already carry a nonce alone', () => {
    expect(addScriptNonce('<script nonce="old">a()</script>', 'N')).toBe(
      '<script nonce="old">a()</script>',
    );
  });

  it('ignores escaped text', () => {
    expect(addScriptNonce('&lt;script&gt;', 'N')).toBe('&lt;script&gt;');
  });
});

describe('buildCspPolicy', () => {
  it('puts the nonce and strict-dynamic in script-src', () => {
    const script = buildCspPolicy('N')
      .split('; ')
      .find((d) => d.startsWith('script-src '));
    expect(script).toMatch(/^script-src 'nonce-N' 'self' 'strict-dynamic'/);
  });

  it('keeps the enforced baseline directives', () => {
    const policy = buildCspPolicy('N');
    for (const d of [
      "form-action 'self'",
      "frame-ancestors 'self'",
      "object-src 'none'",
      "base-uri 'self'",
    ]) {
      expect(policy).toContain(d);
    }
  });

  it('adds report-uri only when given', () => {
    expect(buildCspPolicy('N')).not.toContain('report-uri');
    expect(buildCspPolicy('N', 'https://r.example/x')).toMatch(
      /; report-uri https:\/\/r\.example\/x$/,
    );
  });
});

describe('datadogCspReportUri', () => {
  it('targets the Datadog CSP intake with service tags', () => {
    const url = new URL(datadogCspReportUri('tok', 'staging', '0.1.40'));
    expect(url.origin).toBe('https://browser-intake-datadoghq.com');
    expect(url.searchParams.get('dd-api-key')).toBe('tok');
    expect(url.searchParams.get('ddsource')).toBe('csp-report');
    expect(url.searchParams.get('ddtags')).toBe('service:crowdfunding,env:staging,version:0.1.40');
  });
});
