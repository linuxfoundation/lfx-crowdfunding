// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { InitiativeTypeConfig } from './types/initiative-card.types';

export const initiativeTypeConfigMap: Record<string, InitiativeTypeConfig> = {
  project: {
    label: 'Project',
    icon: 'laptop-code',
    colorClass: 'text-accent-600',
    gradient: 'linear-gradient(180deg, rgba(236, 244, 255, 0.5) 0%, rgba(255, 255, 255, 0.5) 50%)',
  },
  mentorship: {
    label: 'Mentorship',
    icon: 'chalkboard-user',
    colorClass: 'text-neutral-600',
    gradient: 'linear-gradient(180deg, rgba(241, 245, 249, 0.5) 0%, rgba(255, 255, 255, 0.5) 50%)',
  },
  security_audit: {
    label: 'Security Audit',
    icon: 'box-magnifying-glass',
    colorClass: 'text-warning-600',
    gradient: 'linear-gradient(180deg, rgba(255, 251, 235, 0.5) 0%, rgba(255, 255, 255, 0.5) 50%)',
  },
  event: {
    label: 'Event',
    icon: 'calendar-days',
    colorClass: 'text-warning-600',
    gradient: 'linear-gradient(180deg, rgba(255, 251, 235, 0.5) 0%, rgba(255, 255, 255, 0.5) 50%)',
  },
  general_fund: {
    label: 'General Fund',
    icon: 'hand-holding-dollar',
    colorClass: 'text-positive-600',
    gradient: 'linear-gradient(180deg, rgba(240, 253, 244, 0.5) 0%, rgba(255, 255, 255, 0.5) 50%)',
  },
};

export const defaultInitiativeTypeConfig: InitiativeTypeConfig = {
  label: 'Initiative',
  icon: 'stars',
  colorClass: 'text-neutral-600',
  gradient: 'linear-gradient(180deg, rgba(241, 245, 249, 0.5) 0%, rgba(255, 255, 255, 0.5) 50%)',
};
