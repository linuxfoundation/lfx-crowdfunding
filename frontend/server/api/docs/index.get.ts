// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { readdirSync, readFileSync, existsSync } from 'node:fs';
import { join } from 'node:path';
import { getDocsDir, parseFrontmatter, formatDate, toTitleCase } from '../../utils/doc-utils';
import type { DocSection, DocSectionsResponse } from '#shared/types/documentation.types';

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
