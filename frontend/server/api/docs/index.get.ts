// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { readdirSync, readFileSync, existsSync } from 'node:fs';
import { resolve, join } from 'node:path';
import { load as parseYaml } from 'js-yaml';
import type { DocSection, DocSectionsResponse } from '#shared/types/documentation.types';

function getDocsDir(): string {
  return resolve(process.cwd(), '../docs/user');
}

function parseFrontmatter(raw: string): { data: Record<string, unknown>; content: string } {
  const match = raw.match(/^---\r?\n([\s\S]*?)\r?\n---\r?\n?([\s\S]*)$/);
  if (!match) return { data: {}, content: raw };
  const data = (parseYaml(match[1]) ?? {}) as Record<string, unknown>;
  return { data, content: match[2] };
}

export default defineEventHandler((): DocSectionsResponse => {
  const docsDir = getDocsDir();

  if (!existsSync(docsDir)) {
    return { sections: [] };
  }

  const entries = readdirSync(docsDir, { withFileTypes: true });
  const sectionDirs = entries.filter((e) => e.isDirectory());

  const sections: DocSection[] = [];

  for (const dir of sectionDirs) {
    const indexPath = join(docsDir, dir.name, 'index.md');
    if (!existsSync(indexPath)) continue;

    const raw = readFileSync(indexPath, 'utf-8');
    const { data: fm } = parseFrontmatter(raw);

    // Scan sub-directories for child sections
    const subEntries = readdirSync(join(docsDir, dir.name), { withFileTypes: true });
    const children: DocSection[] = [];

    for (const subDir of subEntries.filter((e) => e.isDirectory())) {
      const subIndexPath = join(docsDir, dir.name, subDir.name, 'index.md');
      if (!existsSync(subIndexPath)) continue;

      const subRaw = readFileSync(subIndexPath, 'utf-8');
      const { data: subFm } = parseFrontmatter(subRaw);

      children.push({
        slug: `${dir.name}/${subDir.name}`,
        title: (subFm.title as string | undefined) ?? toTitleCase(subDir.name),
        description: (subFm.description as string | undefined) ?? '',
        displayOrder: (subFm.display_order as number | undefined) ?? 99,
        tags: (subFm.tags as string[] | undefined) ?? [],
        lastUpdated: subFm.last_updated != null ? formatDate(subFm.last_updated) : null,
        children: [],
      });
    }

    children.sort((a, b) => a.displayOrder - b.displayOrder || a.title.localeCompare(b.title));

    sections.push({
      slug: dir.name,
      title: (fm.title as string | undefined) ?? toTitleCase(dir.name),
      description: (fm.description as string | undefined) ?? '',
      displayOrder: (fm.display_order as number | undefined) ?? 99,
      tags: (fm.tags as string[] | undefined) ?? [],
      lastUpdated: fm.last_updated != null ? formatDate(fm.last_updated) : null,
      children,
    });
  }

  sections.sort((a, b) => a.displayOrder - b.displayOrder || a.title.localeCompare(b.title));

  return { sections };
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
