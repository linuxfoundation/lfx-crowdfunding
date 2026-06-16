// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { readFileSync, existsSync } from 'node:fs';
import { resolve, join, sep } from 'node:path';
import { marked, Renderer } from 'marked';
import DOMPurify from 'isomorphic-dompurify';
import {
  getDocsDir,
  parseFrontmatter,
  rewriteDocLink,
  escapeAttr,
  formatDate,
  toTitleCase,
} from '../../utils/doc-utils';
import type { DocArticle } from '#shared/types/documentation.types';

function buildRenderer(slugDir: string): Renderer {
  const renderer = new Renderer();
  renderer.link = ({ href, title, text }) => {
    const rewritten = rewriteDocLink(href ?? '', slugDir);
    const isExternal = rewritten.startsWith('http://') || rewritten.startsWith('https://');
    const attrs = [
      `href="${escapeAttr(rewritten)}"`,
      title ? `title="${escapeAttr(title)}"` : '',
      isExternal ? 'target="_blank" rel="noopener noreferrer"' : '',
    ]
      .filter(Boolean)
      .join(' ');
    return `<a ${attrs}>${text}</a>`;
  };
  return renderer;
}

export default defineEventHandler(async (event): Promise<DocArticle> => {
  const slugParts = getRouterParam(event, 'slug');

  if (!slugParts) {
    throw createError({ statusCode: 400, message: 'Missing slug' });
  }

  const normalised = slugParts.toLowerCase().replace(/^\/|\/$/g, '');
  const safeDocsDir = resolve(getDocsDir());

  // Build candidate paths then guard against path traversal before any disk access
  const candidates = [
    resolve(join(safeDocsDir, `${normalised}.md`)),
    resolve(join(safeDocsDir, normalised, 'index.md')),
  ];

  if (candidates.some((c) => !c.startsWith(safeDocsDir + sep))) {
    throw createError({ statusCode: 400, message: 'Invalid slug' });
  }

  let raw: string | null = null;
  let isIndex = false;
  for (const [i, candidate] of candidates.entries()) {
    if (existsSync(candidate)) {
      raw = readFileSync(candidate, 'utf-8');
      isIndex = i === 1;
      break;
    }
  }

  if (!raw) {
    throw createError({ statusCode: 404, message: `Documentation page not found: ${normalised}` });
  }

  // Directory context for resolving relative links:
  // index.md files (slug = 'donations') → slugDir = 'donations'
  // direct .md files (slug = 'donations/history') → slugDir = 'donations'
  const slugDir = isIndex ? normalised : normalised.split('/').slice(0, -1).join('/');

  const { data: fm, content } = parseFrontmatter(raw);
  const renderedHtml = await marked.parse(content, {
    renderer: buildRenderer(slugDir),
    async: true,
  });
  const bodyHtml = DOMPurify.sanitize(renderedHtml);

  return {
    slug: normalised,
    title:
      (fm.title as string | undefined) ?? toTitleCase(normalised.split('/').pop() ?? normalised),
    description: (fm.description as string | undefined) ?? '',
    bodyHtml,
    tags: (fm.tags as string[] | undefined) ?? [],
    lastUpdated: fm.last_updated != null ? formatDate(fm.last_updated) : null,
  };
});
