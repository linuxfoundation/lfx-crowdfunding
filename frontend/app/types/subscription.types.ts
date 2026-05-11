// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

export interface Budget {
  amount: number;
  allocation?: string;
}

export interface Goal {
  budget: Budget;
}

export interface SubscribableBudget {
  documentation?: { budget: Budget; allocation?: string };
  bugBounty?: { budget: Budget; allocation?: string };
  development?: { budget: Budget; allocation?: string };
  marketing?: { budget: Budget; allocation?: string };
  meetups?: { budget: Budget; allocation?: string };
  travel?: { budget: Budget; allocation?: string };
  mentee?: { budget: Budget; allocation?: string };
  other?: { budget: Budget; allocation?: string };
}

export interface SubscribableEdit extends SubscribableBudget {
  color?: string;
  description?: string;
  industry?: string;
  logo?: File | string;
}

export interface SubscribableDraft extends SubscribableEdit {
  color: string;
  description: string;
  industry: string;
  name: string;
}
