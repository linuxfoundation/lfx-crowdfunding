// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { readdir, readFile } from 'node:fs/promises';
import { join } from 'node:path';
import {
  getDocsDir,
  pathExists,
  parseFrontmatter,
  formatDate,
  toTitleCase,
} from '../../utils/doc-utils';
import type { DocSection, DocSectionsResponse } from '#shared/types/documentation.types';

export default defineEventHandler(async (): Promise<DocSectionsResponse> => {
  const docsDir = await getDocsDir();

  if (!(await pathExists(docsDir))) {
    return { sections: [] };
  }

  const entries = await readdir(docsDir, { withFileTypes: true });
  const sectionDirs = entries.filter((e) => e.isDirectory());

  const sections: DocSection[] = [];

  for (const dir of sectionDirs) {
    const indexPath = join(docsDir, dir.name, 'index.md');
    let raw: string;
    try {
      raw = await readFile(indexPath, 'utf-8');
    } catch {
      continue;
    }
    const { data: fm } = parseFrontmatter(raw);

    // Scan sub-directories for child sections
    const subEntries = await readdir(join(docsDir, dir.name), { withFileTypes: true });
    const children: DocSection[] = [];

    for (const subDir of subEntries.filter((e) => e.isDirectory())) {
      const subIndexPath = join(docsDir, dir.name, subDir.name, 'index.md');
      let subRaw: string;
      try {
        subRaw = await readFile(subIndexPath, 'utf-8');
      } catch {
        continue;
      }
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
