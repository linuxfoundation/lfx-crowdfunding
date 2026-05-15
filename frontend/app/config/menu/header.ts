// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { AppRoute } from '~/config/routes';

export interface HeaderMenuChild {
  label: string;
  icon: string;
  to: string;
}

export interface HeaderMenuItem {
  label: string;
  icon: string;
  to?: string;
  children?: HeaderMenuChild[];
}

export const lfxHeaderMenu: HeaderMenuItem[] = [
  { label: 'Initiatives', icon: 'folder-heart', to: AppRoute.Initiatives },
  { label: 'Statistics', icon: 'chart-pie-simple', to: AppRoute.Statistics },
  {
    label: 'More',
    icon: 'ellipsis',
    children: [
      { label: 'For Projects', icon: 'laptop-code', to: AppRoute.ForProjects },
      { label: 'For Companies', icon: 'buildings', to: AppRoute.ForCompanies },
      { label: 'About', icon: 'circle-info', to: AppRoute.About },
    ],
  },
];
