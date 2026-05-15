// Copyright (c) 2025 The Linux Foundation and each contributor.
// SPDX-License-Identifier: MIT

interface MenuFooter {
  label: string;
  href: string;
}

interface MenuConfig {
  footer: MenuFooter;
}

export const lfxMenu: MenuConfig = {
  footer: {
    label: 'Know more about LFX Platform',
    href: 'https://lfx.linuxfoundation.org',
  },
};
