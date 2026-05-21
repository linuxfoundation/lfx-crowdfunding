// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

interface CategoryVisual {
  color: string;
  icon: string;
}

enum FundingCategoryId {
  SecurityAudit = 'security audit',
  Security = 'security',
  Infrastructure = 'infrastructure',
  Community = 'community',
  Events = 'events',
  Travel = 'travel',
  Mentorship = 'mentorship',
  GeneralFund = 'general fund',
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
  [FundingCategoryId.Mentorship]: { color: '#ec4899', icon: 'graduation-cap' },
  [FundingCategoryId.GeneralFund]: { color: '#f43f5e', icon: 'hand-holding-dollar' },
  [FundingCategoryId.Other]: { color: '#f97316', icon: 'circle-dot' },
  [FundingCategoryId.Uncategorised]: { color: '#14b8a6', icon: 'circle-question' },
  [FundingCategoryId.Uncategorized]: { color: '#14b8a6', icon: 'circle-question' },
};

const FALLBACK: CategoryVisual = { color: '#e2e8f0', icon: 'tag' };

export function getCategoryVisual(name: string): CategoryVisual {
  return CATEGORY_VISUALS[name.toLowerCase() as FundingCategoryId] ?? FALLBACK;
}
