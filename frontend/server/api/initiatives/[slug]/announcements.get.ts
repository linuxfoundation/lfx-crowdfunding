// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { MOCK_ANNOUNCEMENTS } from '#server/mock-data/announcements';
import type { AnnouncementList } from '#shared/types/announcement.types';

// TODO: replace with a proxy call to the Go backend API
export default defineEventHandler((): AnnouncementList => {
  return MOCK_ANNOUNCEMENTS;
});
