// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { existsSync } from 'node:fs';
import { resolve } from 'node:path';
import { load as parseYaml } from 'js-yaml';

export function getDocsDir(): string {
  // Try both common launch points so the server works regardless of CWD.
  // Prefer docs/user (fromRoot) so that when started from the repo root the
  // correct path is chosen immediately. Fall back to ../docs/user (fromFrontend)
  // only when CWD is the frontend/ subdirectory (pnpm dev workflow):
  //   repo root / CI:                process.cwd() = …/repo-root → docs/user is correct
  //   dev (pnpm dev from frontend/): process.cwd() = …/frontend  → ../docs/user is correct
  const fromRoot = resolve(process.cwd(), 'docs/user');
  const fromFrontend = resolve(process.cwd(), '../docs/user');
  return existsSync(fromRoot) ? fromRoot : fromFrontend;
}

export function parseFrontmatter(raw: string): { data: Record<string, unknown>; content: string } {
  const match = raw.match(/^---\r?\n([\s\S]*?)\r?\n---\r?\n?([\s\S]*)$/);
  if (!match) return { data: {}, content: raw };
  const data = (parseYaml(match[1]) ?? {}) as Record<string, unknown>;
  return { data, content: match[2] };
}

// Rewrite relative doc links to absolute /docs/* paths.
// slugDir is the directory context of the current article (set by the route handler):
//   - top-level index.md  (slug = 'donations')                    → slugDir = 'donations'
//   - nested index.md     (slug = 'donations/make-donation')       → slugDir = 'donations/make-donation'
//   - direct .md file     (slug = 'donations/make-donation/guide') → slugDir = 'donations/make-donation'
export function rewriteDocLink(href: string, slugDir: string): string {
  // Anchors and root-relative paths pass through unchanged
  if (href.startsWith('#') || href.startsWith('/')) {
    return href;
  }

  // Any URL scheme (http:, https:, mailto:, tel:, etc.) passes through unchanged
  if (/^[a-z][a-z0-9+.-]*:/i.test(href)) {
    return href;
  }

  let path = href;

  if (path.startsWith('../')) {
    // Walk up the directory stack for each ../ segment — handles multi-level
    const parts = slugDir ? slugDir.split('/') : [];
    while (path.startsWith('../')) {
      parts.pop();
      path = path.slice(3);
    }
    path = parts.length ? `${parts.join('/')}/${path}` : path;
  } else if (path.startsWith('./')) {
    path = slugDir ? `${slugDir}/${path.slice(2)}` : path.slice(2);
  } else if (slugDir) {
    path = `${slugDir}/${path}`;
  }

  const cleaned = path
    .replace(/\/index\.md$/, '')
    .replace(/\.md$/, '')
    .replace(/\/$/, '');

  return `/docs/${cleaned}`;
}

// Escape characters that are unsafe inside HTML attribute values.
// Handles both double-quoted and single-quoted attributes, and prevents
// tag injection via angle brackets.
export function escapeAttr(value: string): string {
  return value
    .replace(/&/g, '&amp;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;');
}

export function formatDate(val: unknown): string {
  if (val instanceof Date) return val.toISOString().slice(0, 10);
  return String(val).slice(0, 10);
}

export function toTitleCase(slug: string): string {
  return slug
    .split('-')
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(' ');
}
