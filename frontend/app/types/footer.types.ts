// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

export type FooterMenuLink =
  | { name: string; link: string; action?: never; intercom?: never }
  | { name: string; action: () => void; link?: never; intercom?: never }
  | { name: string; intercom: true; link?: never; action?: never };

declare global {
  interface Window {
    // Transcend airgap.js API (loaded in setup/head.ts)
    transcend?: { showConsentManager?: () => void };
  }
}

export interface FooterMenuSection {
  title: string;
  links: FooterMenuLink[];
}
