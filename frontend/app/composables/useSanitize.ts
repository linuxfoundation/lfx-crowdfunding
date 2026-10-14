// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT
import DOMPurify from 'isomorphic-dompurify';

// Owner-authored descriptions/announcements only need formatting and links
// (the Self Serve TipTap editor plus legacy imported HTML). DOMPurify's default
// profile also keeps forms, <style>, style="" and external <img>, which would let
// an owner phish or overlay the page (lfx-self-serve-ops#92).
const RICH_TEXT_CONFIG = {
  ALLOWED_TAGS: [
    'p',
    'br',
    'hr',
    'strong',
    'b',
    'em',
    'i',
    'u',
    's',
    'del',
    'strike',
    'h1',
    'h2',
    'h3',
    'h4',
    'h5',
    'h6',
    'ul',
    'ol',
    'li',
    'blockquote',
    'code',
    'pre',
    'a',
  ],
  ALLOWED_ATTR: ['href'],
  ALLOWED_URI_REGEXP: /^(?:https?:|mailto:)/i,
};

// Links in owner content open in a new tab, without referrer/opener or SEO credit.
const hardenLink = (node: Element) => {
  if (node.tagName === 'A') {
    node.setAttribute('target', '_blank');
    node.setAttribute('rel', 'noopener noreferrer nofollow');
  }
};

// The hook is added per call because the DOMPurify instance is shared with
// sanitize() and the docs route, whose links must stay untouched.
const sanitizeRichText = (html: string): string => {
  DOMPurify.addHook('afterSanitizeAttributes', hardenLink);
  try {
    return DOMPurify.sanitize(html, RICH_TEXT_CONFIG);
  } finally {
    DOMPurify.removeHook('afterSanitizeAttributes');
  }
};

export const useSanitize = () => {
  const sanitize = (dirty: string) => DOMPurify.sanitize(dirty);

  /**
   * Strips all HTML tags from a string, returning plain text suitable for
   * display in contexts where no markup should appear (e.g. card previews).
   */
  const stripHtml = (html: string): string =>
    DOMPurify.sanitize(html, { ALLOWED_TAGS: [], ALLOWED_ATTR: [] });

  /**
   * Sanitizes a description and returns a safe HTML string for use with v-html.
   * If the input is plain text (no HTML tags), converts newlines to paragraph
   * and line-break elements so whitespace is preserved when rendered.
   */
  const escapeHtml = (text: string): string =>
    text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');

  const renderDescription = (raw: string): string => {
    const hasHtmlTags = /<[a-z][\s\S]*>/i.test(raw);
    if (hasHtmlTags) {
      return sanitizeRichText(raw);
    }
    // Plain text: normalize line endings (CRLF/CR → LF), split blank lines into
    // paragraphs, single newlines become <br>. Escape HTML-sensitive characters
    // before wrapping so they survive the HTML parser before DOMPurify runs.
    const html = raw
      .replace(/\r\n?/g, '\n')
      .split(/\n\n+/)
      .filter((p) => p.trim())
      .map((p) => `<p>${escapeHtml(p).replace(/\n/g, '<br>')}</p>`)
      .join('');
    return sanitizeRichText(html || raw);
  };

  return { sanitize, stripHtml, renderDescription };
};
