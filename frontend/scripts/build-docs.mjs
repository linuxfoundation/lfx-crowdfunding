// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

/**
 * Build script: scans docs/user/ and writes a search index to
 * public/assets/docs/search-index.json for use by the client-side search component.
 *
 * Run via: pnpm docs:build
 * Auto-runs before: pnpm dev, pnpm build
 */

import { readdirSync, readFileSync, writeFileSync, existsSync, mkdirSync } from 'node:fs';
import { resolve, join } from 'node:path';
import { load as parseYaml } from 'js-yaml';

const DOCS_DIR = resolve(import.meta.dirname, '../../docs/user');
const OUT_FILE = resolve(import.meta.dirname, '../public/assets/docs/search-index.json');

function parseFrontmatter(raw) {
  const match = raw.match(/^---\r?\n([\s\S]*?)\r?\n---\r?\n?([\s\S]*)$/);
  if (!match) return { data: {}, content: raw };
  const data = /** @type {Record<string, unknown>} */ (parseYaml(match[1]) ?? {});
  return { data, content: match[2] };
}

function stripMarkdown(md) {
  return md
    .replace(/!\[.*?\]\(.*?\)/g, '') // images
    .replace(/\[([^\]]+)\]\(.*?\)/g, '$1') // links → text only
    .replace(/`{1,3}[^`\n]*`{1,3}/g, '') // inline code
    .replace(/^```[\s\S]*?^```/gm, '') // fenced code blocks
    .replace(/^#+\s+/gm, '') // headings
    .replace(/[*_~]{1,3}([^*_~\n]+)[*_~]{1,3}/g, '$1') // bold / italic / strikethrough
    .replace(/^\s*[-*+>|]\s*/gm, '') // lists, blockquotes, table pipes
    .replace(/\n{2,}/g, ' ')
    .replace(/\s+/g, ' ')
    .trim();
}

/**
 * @param {string} dir
 * @param {string} parentSlug
 * @returns {Array<{slug: string, title: string, description: string, content: string}>}
 */
function walk(dir, parentSlug = '') {
  const docs = [];

  if (!existsSync(dir)) return docs;

  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    if (!entry.isDirectory()) continue;

    const slug = parentSlug ? `${parentSlug}/${entry.name}` : entry.name;
    const indexPath = join(dir, entry.name, 'index.md');

    if (!existsSync(indexPath)) continue;

    const raw = readFileSync(indexPath, 'utf-8');
    const { data: fm, content } = parseFrontmatter(raw);

    docs.push({
      slug,
      title: String(fm.title ?? entry.name),
      description: String(fm.description ?? ''),
      content: stripMarkdown(content),
    });

    // recurse into sub-sections
    docs.push(...walk(join(dir, entry.name), slug));
  }

  return docs;
}

const docs = walk(DOCS_DIR);

const outDir = resolve(OUT_FILE, '..');
if (!existsSync(outDir)) mkdirSync(outDir, { recursive: true });

writeFileSync(OUT_FILE, JSON.stringify(docs));

process.stdout.write(
  `✓ docs:build — ${docs.length} documents → public/assets/docs/search-index.json\n`,
);
