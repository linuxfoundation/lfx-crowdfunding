// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { FeaturedInitiativesResponse } from '#shared/types/static-pages.types';

export const MOCK_FEATURED_INITIATIVES: FeaturedInitiativesResponse = {
  data: [
    {
      id: 'kubernetes-security-audit',
      name: 'Kubernetes Security Audit',
      raisedCents: 20_000_000,
      goalCents: 54_670_000,
      supporterCount: 100,
    },
    {
      id: 'lets-encrypt-infrastructure',
      name: "Let's Encrypt Infrastructure",
      raisedCents: 31_015_000,
      goalCents: 50_000_000,
      supporterCount: 87,
    },
    {
      id: 'linux-kernel-mentorship',
      name: 'Linux Kernel Mentorship',
      raisedCents: 18_295_000,
      goalCents: 50_000_000,
      supporterCount: 64,
    },
    {
      id: 'rust-for-linux',
      name: 'Rust for Linux',
      raisedCents: 25_355_000,
      goalCents: 50_000_000,
      supporterCount: 142,
    },
    {
      id: 'openssf-scorecard',
      name: 'OpenSSF Scorecard',
      raisedCents: 12_380_000,
      goalCents: 50_000_000,
      supporterCount: 53,
    },
    {
      id: 'security-portfolio',
      name: 'Security Portfolio',
      raisedCents: 39_860_000,
      goalCents: 50_000_000,
      supporterCount: 215,
    },
  ],
};
