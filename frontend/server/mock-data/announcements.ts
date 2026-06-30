// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { AnnouncementList } from '#shared/types/announcement.types';

export const MOCK_ANNOUNCEMENTS: AnnouncementList = {
  data: [
    {
      id: '1',
      title: 'Spring 2026 Cohort Applications Open',
      body: 'Applications for the Spring 2026 mentorship cohort are now open. Apply by April 15th.',
      publishedAt: '2026-03-15T00:00:00Z',
    },
    {
      id: '2',
      title: 'Fall 2025 Cohort Results',
      body: '12 mentees successfully completed the program, with 6 landing their first kernel patch.',
      publishedAt: '2026-01-10T00:00:00Z',
    },
    {
      id: '3',
      title: 'New Funding Milestone Reached',
      body: 'We have reached $500K in total funding. Thank you to all our sponsors and individual donors.',
      publishedAt: '2025-11-20T00:00:00Z',
    },
  ],
  totalCount: 3,
};
