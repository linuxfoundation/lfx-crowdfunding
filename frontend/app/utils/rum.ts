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

const HYDRATION_WARNING_PREFIX = '[Vue warn]: Hydration';
const MAX_ELEMENT_HTML = 300;

/**
 * Builds a one-line message from a Vue hydration-mismatch console.warn, or null for any other
 * warning. RUM only ingests console.error, and Vue logs just "Hydration completed but contains
 * mismatches." there; the node and the server/client values only go to console.warn.
 */
export function hydrationWarningMessage(args: unknown[]): string | null {
  if (typeof args[0] !== 'string' || !args[0].startsWith(HYDRATION_WARNING_PREFIX)) return null;
  return args
    .map((arg) => {
      if (typeof arg === 'string') return arg;
      if (arg && typeof (arg as Element).outerHTML === 'string') {
        // Opening tag only: the node's children can be large and may hold user content.
        const html = (arg as Element).outerHTML;
        const openTag = html.slice(0, html.indexOf('>') + 1) || html;
        return openTag.slice(0, MAX_ELEMENT_HTML);
      }
      return String(arg);
    })
    .join(' ')
    .trim();
}
