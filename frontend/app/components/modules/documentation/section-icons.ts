// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Topic-associated icons keyed by documentation section slug.
// Shared by the docs landing cards and the docs sidebar so both stay in sync.
const SECTION_ICONS: Record<string, string> = {
  'getting-started': 'book-open',
  initiatives: 'folder-heart',
  donations: 'circle-dollar-to-slot',
  'payment-account': 'credit-card',
  backers: 'hands-holding-dollar',
  reimbursements: 'memo-circle-check',
};

const DEFAULT_SECTION_ICON = 'file-lines';

export function getSectionIcon(slug: string): string {
  return SECTION_ICONS[slug] ?? DEFAULT_SECTION_ICON;
}
