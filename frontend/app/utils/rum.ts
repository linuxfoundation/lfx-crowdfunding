// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { RumEvent } from '@datadog/browser-rum';

/**
 * True for an 'eval' CSP violation with no source file. Our bundle never evals; these come
 * from scripts the browser injects itself (e.g. HeyTapBrowser). The SDK only builds a stack
 * when the report names a source file, so an eval from our own scripts still gets reported.
 */
export function isInjectedEvalCspReport(event: RumEvent): boolean {
  return (
    event.type === 'error' &&
    event.error.source === 'report' &&
    event.error.message.startsWith("csp_violation: 'eval'") &&
    !event.error.stack
  );
}
