// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

export interface FundraiseSubmission {
  initiativeType: string;
  details: {
    name: string;
    description: string;
    githubUrl?: string;
    tags?: string;
    auditScope?: string;
    eventDate?: string;
    location?: string;
    eventbriteUrl?: string;
  };
  goals: {
    goalAmountCents: number;
    deadline?: string;
    expectedAttendees?: string;
  } | null;
}

export interface FundraiseRecord extends FundraiseSubmission {
  id: string;
  status: 'pending_review';
  createdAt: string;
}
