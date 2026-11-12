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

const PAGE_LANG = 'en';
const TRANSLATED_CLASSES = ['translated-ltr', 'translated-rtl'];

/**
 * True when the browser has machine-translated the page. Chrome Translate adds a
 * translated-ltr/-rtl class to <html> and sets its lang to the target language; other
 * translators change lang too. The page itself is always rendered in English.
 */
export function isTranslatedPage(root: Pick<Element, 'classList' | 'getAttribute'>): boolean {
  if (TRANSLATED_CLASSES.some((name) => root.classList.contains(name))) return true;
  const lang = root.getAttribute('lang')?.toLowerCase();
  return !!lang && lang !== PAGE_LANG && !lang.startsWith(`${PAGE_LANG}-`);
}

const HYDRATION_WARNING_PREFIX = '[Vue warn]: Hydration';
const MAX_ELEMENT_HTML = 300;
const REDACTED = '[redacted]';
// Attributes whose values are our own markup, not user content. src is already sent to RUM as a
// resource URL. Every other attribute (value, alt, title, aria-label, href...) is redacted.
const SAFE_ATTRIBUTES = new Set(['id', 'class', 'style', 'src', 'type', 'role', 'width', 'height']);
// Mismatch kinds whose server/client values are formatted as key="value" (Vue's propHasMismatch).
const ATTRIBUTE_MISMATCH = /^\[Vue warn\]: Hydration (attribute|class|style) mismatch/;
const VALUE_LINE = /^(\s*-?\s*(?:rendered on server|expected on client):)\s?(.*)$/;
const ATTRIBUTE_VALUE = /^([\w:.@-]+)="([\s\S]*)"$/;
const TAG_NAME = /^[a-z][a-z0-9]*(-[a-z0-9]+)*$/;
// Fixed text Vue adds to mismatch warnings; anything else in a string argument is redacted.
const SAFE_LINES = [
  /^Server rendered element contains (more|fewer) child nodes than client vdom\.$/,
  /^Note: this mismatch is check-only\./,
  /^You should fix the source of the mismatch\.$/,
  /^\((text|start of fragment)\)$/,
];

interface WarnNode {
  nodeName: string;
  attributes?: ArrayLike<{ name: string; value: string }>;
}

function isWarnNode(arg: unknown): arg is WarnNode {
  return !!arg && typeof arg === 'object' && typeof (arg as WarnNode).nodeName === 'string';
}

function safeAttribute(name: string, value: string): string {
  return `${name}="${SAFE_ATTRIBUTES.has(name.toLowerCase()) ? value : REDACTED}"`;
}

/** Opening tag rebuilt from the node's attributes, with non-allowlisted values redacted. */
function describeNode(node: WarnNode): string {
  if (!node.attributes) return node.nodeName.toLowerCase(); // #text, #comment
  const attrs = Array.from(node.attributes, ({ name, value }) => ` ${safeAttribute(name, value)}`);
  return `<${node.nodeName.toLowerCase()}${attrs.join('')}>`.slice(0, MAX_ELEMENT_HTML);
}

function describeValue(value: string, isAttributeMismatch: boolean): string {
  if (!value || value === '(not rendered)') return value;
  const attribute = isAttributeMismatch ? ATTRIBUTE_VALUE.exec(value) : null;
  return attribute ? safeAttribute(attribute[1]!, attribute[2]!) : REDACTED;
}

/** Keeps Vue's fixed wording and the allowlisted values; redacts page text. */
function describeText(text: string, isAttributeMismatch: boolean): string {
  // A node mismatch passes the expected element's tag name as its own argument.
  if (TAG_NAME.test(text)) return text;
  const lines = text.split('\n').map((line) => {
    const valueLine = VALUE_LINE.exec(line);
    if (valueLine) {
      const value = describeValue(valueLine[2]!, isAttributeMismatch);
      return value ? `${valueLine[1]} ${value}` : valueLine[1]!;
    }
    const trimmed = line.trim();
    return !trimmed || SAFE_LINES.some((re) => re.test(trimmed)) ? line : REDACTED;
  });
  // Multi-line page text redacts to one marker, not one per line.
  return lines.filter((line, i) => line !== REDACTED || lines[i - 1] !== REDACTED).join('\n');
}

function describeArg(arg: unknown, isAttributeMismatch: boolean): string {
  if (typeof arg === 'string') return describeText(arg, isAttributeMismatch);
  if (isWarnNode(arg)) return describeNode(arg);
  if (typeof arg === 'symbol' || typeof arg === 'number' || typeof arg === 'boolean') {
    return String(arg);
  }
  // Node mismatches pass the client vnode type: a component object or a tag name string.
  if (arg && typeof arg === 'object') {
    const { __name, name } = arg as { __name?: unknown; name?: unknown };
    const label = typeof __name === 'string' ? __name : typeof name === 'string' ? name : null;
    return label ? `<${label}>` : '(object)';
  }
  return String(arg);
}

/**
 * Builds a one-line message from a Vue hydration-mismatch console.warn, or null for any other
 * warning. RUM only ingests console.error, and Vue logs just "Hydration completed but contains
 * mismatches." there; the node and the server/client values only go to console.warn.
 *
 * The message goes to a third party and session-replay masking doesn't cover addError, so page
 * text and attribute values outside SAFE_ATTRIBUTES are redacted. Vue's own wording in the first
 * argument is kept as is.
 */
export function hydrationWarningMessage(args: unknown[]): string | null {
  const [first, ...rest] = args;
  if (typeof first !== 'string' || !first.startsWith(HYDRATION_WARNING_PREFIX)) return null;
  const isAttributeMismatch = ATTRIBUTE_MISMATCH.test(first);
  return [first, ...rest.map((arg) => describeArg(arg, isAttributeMismatch))].join(' ').trim();
}
