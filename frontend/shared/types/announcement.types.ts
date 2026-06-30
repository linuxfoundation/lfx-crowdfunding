// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

export interface Announcement {
  id: string;
  title: string;
  body: string;
  publishedAt: string; // ISO date string
}

export interface AnnouncementList {
  data: Announcement[];
  totalCount: number;
}
