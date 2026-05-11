// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

export interface ProjectFundingStatus {
  totalAnnualGoalInCents?: number;
  totalDonationsInCents: number;
  totalSubscriptionCount: number;
  annualSubscriptionAmountInCents: number;
  annualSubscriptionRemainingAmountInCents: number;
}
