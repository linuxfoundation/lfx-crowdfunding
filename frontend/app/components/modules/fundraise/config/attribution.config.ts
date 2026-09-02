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
    emptyMessage: "You aren't affiliated with any organization yet.",
    escapeHatchLabel: "Can't find your organization? Manage",
  },
  {
    value: 'project',
    label: 'Project/Foundation',
    description: 'For a fundraiser sponsored by a project or foundation you participate in.',
    entityLabel: 'Select Project/Foundation',
    emptyMessage: "You aren't affiliated with any project or foundation yet.",
    escapeHatchLabel: "Can't find your Project or Foundation? Manage",
  },
];

// Self Serve path for the "Work History & Affiliations" escape hatch, appended to
// runtimeConfig.public.selfServeUrl. Best-guess pending confirmation — see PR description.
export const AFFILIATIONS_MANAGEMENT_PATH = '/profile/work-history';

export const createDefaultAttribution = (): AttributionData => ({
  kind: 'personal',
  entityId: null,
});
