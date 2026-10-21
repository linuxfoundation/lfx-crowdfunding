// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { AttributionData, AttributionOption } from '~/types/fundraise.types';

export const ATTRIBUTION_OPTIONS: AttributionOption[] = [
  {
    value: 'personal',
    label: 'Personal',
    description: 'Create this initiative under your own name.',
  },
  {
    value: 'organization',
    label: 'Organization',
    description: 'For a fundraiser sponsored by an organization you work for.',
    entityLabel: 'Select organization',
    emptyMessage: "You don't have a role in any organization yet.",
    escapeHatchLabel:
      'Only organizations where you have a role in LFX are listed. Ask an admin of your organization to add you in',
  },
  {
    value: 'project',
    label: 'Project/Foundation',
    description: 'For a fundraiser sponsored by a project or foundation you participate in.',
    entityLabel: 'Select Project/Foundation',
    emptyMessage: "You don't have a role in any project or foundation yet.",
    escapeHatchLabel:
      "Only projects and foundations you've been granted access to in LFX are listed. Ask a project admin to add you in",
  },
];

export const createDefaultAttribution = (): AttributionData => ({
  kind: 'personal',
  entityId: null,
});
