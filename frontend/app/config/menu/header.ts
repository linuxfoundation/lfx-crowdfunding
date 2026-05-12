// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

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
  { label: 'Initiatives', icon: 'folder-heart', to: '/initiatives' },
  { label: 'Statistics', icon: 'chart-pie-simple', to: '/statistics' },
  {
    label: 'More',
    icon: 'ellipsis',
    children: [
      { label: 'For Projects', icon: 'laptop-code', to: '/for-projects' },
      { label: 'For Companies', icon: 'buildings', to: '/for-companies' },
      { label: 'About', icon: 'circle-info', to: '/about' },
    ],
  },
];
