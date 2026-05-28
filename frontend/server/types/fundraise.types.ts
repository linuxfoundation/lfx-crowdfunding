// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Wire shapes for POST /api/fundraise → POST /v1/initiatives

export interface FundraiseContactInput {
  firstName: string;
  lastName: string;
  email: string;
  phone: string;
  preferredContact: 'email' | 'phone';
}

export interface FundraiseBeneficiaryInput {
  name: string;
  email: string;
}

export interface ProjectFundraisePayload {
  initiativeType: 'project';
  name: string;
  description: string;
  websiteUrl?: string;
  cocUrl?: string;
  repositoryUrl?: string;
  beneficiaries?: FundraiseBeneficiaryInput[];
  annualFundingGoalCents?: number;
}

export interface SecurityAuditFundraisePayload {
  initiativeType: 'security_audit';
  name: string;
  description: string;
  websiteUrl?: string;
  repositoryUrl?: string;
  fundingGoalCents?: number;
  primaryContact?: FundraiseContactInput;
  secondaryContact?: FundraiseContactInput;
  technicalLead?: FundraiseContactInput;
}

export interface EventFundraisePayload {
  initiativeType: 'event';
  name: string;
  description: string;
  websiteUrl?: string;
  registrationUrl?: string;
  startDate?: string;
  endDate?: string;
  city?: string;
  country?: string;
  isOnline?: boolean;
  beneficiaries?: FundraiseBeneficiaryInput[];
  sponsorshipGoalCents?: number;
}

export interface GeneralFundFundraisePayload {
  initiativeType: 'general_fund';
  name: string;
  description: string;
  websiteUrl?: string;
  beneficiaries?: FundraiseBeneficiaryInput[];
  annualFundingGoalCents?: number;
}

export type FundraisePayload =
  | ProjectFundraisePayload
  | SecurityAuditFundraisePayload
  | EventFundraisePayload
  | GeneralFundFundraisePayload;

export interface FundraiseResult {
  id: string;
  slug: string;
  name: string;
  status: string;
}
