// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { AppRoute } from '~/config/routes';

export interface NavCard {
  id: string;
  label: string;
  icon: string;
  tagline: string;
  href: string;
  gradient: string;
}

export const navCards: NavCard[] = [
  {
    id: 'statistics',
    label: 'Statistics',
    icon: 'chart-pie-simple',
    tagline: '$5.8M funds raised by 1,842 supporters',
    href: AppRoute.Statistics,
    gradient: 'linear-gradient(180deg, #ffffff 0%, #ecf4ff 100%)',
  },
  {
    id: 'for-projects',
    label: 'For projects',
    icon: 'laptop-code',
    tagline: 'Raise funds for your open source project',
    href: AppRoute.ForProjects,
    gradient: 'linear-gradient(180deg, #ffffff 0%, #f1f5f9 100%)',
  },
  {
    id: 'for-companies',
    label: 'For companies',
    icon: 'buildings',
    tagline: 'Invest in the open source your business depends on',
    href: AppRoute.ForCompanies,
    gradient: 'linear-gradient(180deg, #ffffff 0%, #ede9fe 100%)',
  },
];
