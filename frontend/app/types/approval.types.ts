// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

export type ApprovalAction = 'approve' | 'decline';

export interface ApprovalResult {
  id: string;
  name: string;
  slug: string;
  status: string;
}
