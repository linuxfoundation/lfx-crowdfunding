// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { formatTitleCase } from '~/utils/formatter';

interface CategoryVisual {
  color: string;
  icon: string;
  // Tailwind text-color class for the icon; defaults to white when unset.
  iconColor?: string;
}

enum FundingCategoryId {
  SecurityAudit = 'security audit',
  Security = 'security',
  Infrastructure = 'infrastructure',
  Community = 'community',
  Events = 'events',
  Travel = 'travel',
  Development = 'development',
  Documentation = 'documentation',
  Mentorship = 'mentorship',
  GeneralFund = 'general fund',
  Marketing = 'marketing',
  Meetups = 'meetups',
  BugBounty = 'bugbounty',
  Other = 'other',
  Uncategorised = 'uncategorised',
  Uncategorized = 'uncategorized',
}

const CATEGORY_VISUALS: Record<FundingCategoryId, CategoryVisual> = {
  [FundingCategoryId.SecurityAudit]: { color: '#fe9a00', icon: 'shield' },
  [FundingCategoryId.Security]: { color: '#fe9a00', icon: 'shield' },
  [FundingCategoryId.Infrastructure]: { color: '#6366f1', icon: 'server' },
  [FundingCategoryId.Community]: { color: '#009aff', icon: 'users' },
  [FundingCategoryId.Events]: { color: '#00bc7d', icon: 'calendar' },
  [FundingCategoryId.Travel]: { color: '#8e51ff', icon: 'plane' },
  [FundingCategoryId.Development]: { color: '#009aff', icon: 'code' },
  [FundingCategoryId.Documentation]: { color: '#6b7280', icon: 'book-open' },
  [FundingCategoryId.Mentorship]: { color: '#4b5563', icon: 'graduation-cap' },
  [FundingCategoryId.GeneralFund]: { color: '#00bc7d', icon: 'hand-holding-dollar' },
  [FundingCategoryId.Marketing]: { color: '#ef4444', icon: 'bullhorn' },
  [FundingCategoryId.Meetups]: { color: '#14b8a6', icon: 'handshake' },
  [FundingCategoryId.BugBounty]: { color: '#eab308', icon: 'bug' },
  [FundingCategoryId.Other]: { color: '#f97316', icon: 'circle-dot' },
  [FundingCategoryId.Uncategorised]: { color: '#14b8a6', icon: 'circle-question' },
  [FundingCategoryId.Uncategorized]: { color: '#14b8a6', icon: 'circle-question' },
};

// "Others" and any unmapped category use this: a light-gray bubble needs a darker icon to stay legible.
const FALLBACK: CategoryVisual = { color: '#e2e8f0', icon: 'tag', iconColor: 'text-neutral-500' };

// Display-name overrides for categories whose label should differ from the raw name.
const CATEGORY_LABELS: Partial<Record<FundingCategoryId, string>> = {
  [FundingCategoryId.GeneralFund]: 'General Funds',
};

export function getCategoryVisual(name: string): CategoryVisual {
  return CATEGORY_VISUALS[name.toLowerCase() as FundingCategoryId] ?? FALLBACK;
}

export function getCategoryLabel(name: string): string {
  return CATEGORY_LABELS[name.toLowerCase() as FundingCategoryId] ?? formatTitleCase(name);
}
