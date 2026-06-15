// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT
import DOMPurify from 'isomorphic-dompurify';

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
      return DOMPurify.sanitize(raw);
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
    return DOMPurify.sanitize(html || raw);
  };

  return { sanitize, stripHtml, renderDescription };
};
