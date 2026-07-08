// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

export interface Announcement {
  id: string;
  initiativeId: string;
  createdBy: string;
  title: string;
  description: string;
  createdOn: string;
  updatedOn: string;
}

export interface AnnouncementList {
  data: Announcement[];
  totalCount: number;
}
