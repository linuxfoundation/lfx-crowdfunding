// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { Tab } from '~/components/uikit/tabs/types/tab.types';

export const INITIATIVE_FILTER_TABS: Tab[] = [
  { value: 'all', label: 'All', icon: 'grid-round-2' },
  { value: 'project', label: 'Projects', icon: 'laptop-code' },
  { value: 'mentorship', label: 'Mentorships', icon: 'chalkboard-user' },
  { value: 'security_audit', label: 'Security Audits', icon: 'box-magnifying-glass' },
  { value: 'event', label: 'Events', icon: 'calendar' },
  { value: 'general_fund', label: 'General Funds', icon: 'piggy-bank' },
];

export interface SortOption {
  value: string;
  label: string;
}

export const DEFAULT_SORT_OPTION: SortOption = { value: 'supporters', label: 'Most supporters' };

export const INITIATIVE_SORT_OPTIONS: SortOption[] = [
  DEFAULT_SORT_OPTION,
  { value: 'total_raised', label: 'Most funded' },
  { value: 'created_on', label: 'Most recent' },
  { value: 'name', label: 'Name (A–Z)' },
];
