// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

interface CategoryVisual {
  color: string;
  icon: string;
}

const CATEGORY_VISUALS: Record<string, CategoryVisual> = {
  'security audit': { color: '#fe9a00', icon: 'shield' },
  security: { color: '#fe9a00', icon: 'shield' },
  infrastructure: { color: '#6366f1', icon: 'server' },
  community: { color: '#009aff', icon: 'users' },
  events: { color: '#00bc7d', icon: 'calendar' },
  travel: { color: '#8e51ff', icon: 'plane' },
  mentorship: { color: '#ec4899', icon: 'graduation-cap' },
  'general fund': { color: '#f43f5e', icon: 'hand-holding-dollar' },
  other: { color: '#f97316', icon: 'circle-dot' },
  uncategorised: { color: '#14b8a6', icon: 'circle-question' },
  uncategorized: { color: '#14b8a6', icon: 'circle-question' },
};

const FALLBACK: CategoryVisual = { color: '#e2e8f0', icon: 'tag' };

export function getCategoryVisual(name: string): CategoryVisual {
  return CATEGORY_VISUALS[name.toLowerCase()] ?? FALLBACK;
}
