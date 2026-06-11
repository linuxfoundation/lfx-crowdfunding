// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT
import DOMPurify from 'isomorphic-dompurify';

export const useSanitize = () => {
  const sanitize = (dirty: string) => DOMPurify.sanitize(dirty);

  /**
   * Strips all HTML tags from a string, returning plain text suitable for
   * display in contexts where no markup should appear (e.g. card previews).
   */
  const stripHtml = (html: string): string => html.replace(/<[^>]*>/g, '');

  /**
   * Sanitizes an HTML description and returns a safe string for use with v-html.
   */
  const renderDescription = (raw: string): string => DOMPurify.sanitize(raw);

  return { sanitize, stripHtml, renderDescription };
};
