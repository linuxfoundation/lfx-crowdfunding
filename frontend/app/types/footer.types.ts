// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

export interface FooterMenuLink {
  name: string;
  link?: string;
  action?: () => void;
}

export interface FooterMenuSection {
  title: string;
  links: FooterMenuLink[];
}
