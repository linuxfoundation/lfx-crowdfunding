// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

export type FooterMenuLink =
  | { name: string; link: string; action?: never }
  | { name: string; action: () => void; link?: never };

export interface FooterMenuSection {
  title: string;
  links: FooterMenuLink[];
}
