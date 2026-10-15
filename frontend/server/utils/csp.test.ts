// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { describe, it, expect } from 'vitest';
import { buildCspPolicy, datadogCspReportUri } from './csp';

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
