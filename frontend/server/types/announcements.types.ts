// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

export interface BackendAnnouncement {
  id: string;
  initiative_id: string;
  created_by: string;
  title: string;
  description: string;
  created_on: string;
  updated_on: string;
}

export interface BackendAnnouncementList {
  data: BackendAnnouncement[];
  meta: { total: number; limit: number; offset: number };
}
