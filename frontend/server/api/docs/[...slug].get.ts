// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { readFileSync, existsSync } from 'node:fs';
import { resolve, join } from 'node:path';
import { load as parseYaml } from 'js-yaml';
import { marked, Renderer } from 'marked';
import DOMPurify from 'isomorphic-dompurify';
import type { DocArticle } from '#shared/types/documentation.types';

// Rewrite relative doc links to absolute /docs/* paths.
// slugDir is the directory context of the current article:
//   - for index.md files (slug = 'donations') → slugDir = 'donations'
//   - for direct .md files (slug = 'donations/history') → slugDir = 'donations'
function rewriteDocLink(href: string, slugDir: string): string {
  if (
    href.startsWith('#') ||
    href.startsWith('/') ||
    href.startsWith('http://') ||
    href.startsWith('https://')
  ) {
    return href;
  }

  let path = href;

  if (path.startsWith('./')) {
    // Same-directory: ./make-donation/ → donations/make-donation
    path = slugDir ? `${slugDir}/${path.slice(2)}` : path.slice(2);
  } else if (path.startsWith('../')) {
    // Parent-relative: ../initiatives/ → initiatives
    const parentDir = slugDir.includes('/') ? slugDir.split('/').slice(0, -1).join('/') : '';
    path = parentDir ? `${parentDir}/${path.slice(3)}` : path.slice(3);
  } else if (slugDir) {
    // Bare relative (no prefix): treat as same-directory
    path = `${slugDir}/${path}`;
  }

  const cleaned = path
    .replace(/\/index\.md$/, '')
    .replace(/\.md$/, '')
    .replace(/\/$/, '');

  return `/docs/${cleaned}`;
}

function buildRenderer(slugDir: string): Renderer {
  const renderer = new Renderer();
  renderer.link = ({ href, title, text }) => {
    const rewritten = rewriteDocLink(href ?? '', slugDir);
    const isExternal = rewritten.startsWith('http://') || rewritten.startsWith('https://');
    const attrs = [
      `href="${rewritten}"`,
      title ? `title="${title}"` : '',
      isExternal ? 'target="_blank" rel="noopener noreferrer"' : '',
    ]
      .filter(Boolean)
      .join(' ');
    return `<a ${attrs}>${text}</a>`;
  };
  return renderer;
}

function getDocsDir(): string {
  return resolve(process.cwd(), '../docs/user');
}

function parseFrontmatter(raw: string): { data: Record<string, unknown>; content: string } {
  const match = raw.match(/^---\r?\n([\s\S]*?)\r?\n---\r?\n?([\s\S]*)$/);
  if (!match) return { data: {}, content: raw };
  const data = (parseYaml(match[1]) ?? {}) as Record<string, unknown>;
  return { data, content: match[2] };
}

export default defineEventHandler(async (event): Promise<DocArticle> => {
  const slugParts = getRouterParam(event, 'slug');

  if (!slugParts) {
    throw createError({ statusCode: 400, message: 'Missing slug' });
  }

  const normalised = slugParts.toLowerCase().replace(/^\/|\/$/g, '');
  const docsDir = getDocsDir();

  // Try direct .md file first, then index.md inside a folder
  const candidates = [join(docsDir, `${normalised}.md`), join(docsDir, normalised, 'index.md')];

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

function formatDate(val: unknown): string {
  if (val instanceof Date) return val.toISOString().slice(0, 10);
  return String(val).slice(0, 10);
}

function toTitleCase(slug: string): string {
  return slug
    .split('-')
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(' ');
}
