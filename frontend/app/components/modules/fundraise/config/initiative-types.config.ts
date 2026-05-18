// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { InitiativeType } from '~/types/fundraise.types';

export interface InitiativeTypeConfig {
  label: string;
  icon: string;
  reviewMessage: string;
}

export const INITIATIVE_TYPE_CONFIG: Record<InitiativeType, InitiativeTypeConfig> = {
  project: {
    label: 'Project',
    icon: 'code',
    reviewMessage: 'Our team will review your project within 2 business days.',
  },
  security_audit: {
    label: 'OSTIF Security Audit',
    icon: 'box-magnifying-glass',
    reviewMessage: 'Our team will review your security audit within 2 business days.',
  },
  general_fund: {
    label: 'General Fund',
    icon: 'piggy-bank',
    reviewMessage: 'Our team will review your general fund within 2 business days.',
  },
  event: {
    label: 'Event / Meetup',
    icon: 'calendar',
    reviewMessage: 'Our team will review your event within 2 business days.',
  },
};
