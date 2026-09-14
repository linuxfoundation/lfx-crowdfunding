// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Wire shapes for POST /api/fundraise → POST /v1/initiatives

export interface GoalItemInput {
  category: string;
  label: string;
  description: string;
  enabled: boolean;
  percentage: number;
}

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

export type SponsorshipTierInput = {
  name: string;
  enabled: boolean;
  goalCents?: number;
  benefits: string[];
};

// Matches the backend contract in docs/sponsorship-tiers-backend-requirements.md
// (donation_mode + sponsorship_tiers[]) — not yet implemented server-side.
export interface DonationOptionsInput {
  mode: 'tiers' | 'open';
  tiers: SponsorshipTierInput[];
}

// Matches backend §5.0 of docs/rewrite/07-frontend-initiatives-api-guide.md
// (LFXV2-2956): wire shape is `attribution: { type, entity_uid }`, mapped in
// buildBackendPayload. entity_uid must be a UUID.
export interface AttributionInput {
  kind: 'organization' | 'project';
  entityId: string;
}

export interface ProjectFundraisePayload {
  initiativeType: 'project';
  name: string;
  description: string;
  industry?: string;
  websiteUrl?: string;
  ciiProjectId?: string;
  cocUrl?: string;
  repositoryUrl?: string;
  logoUrl?: string;
  beneficiaries?: FundraiseBeneficiaryInput[];
  annualFundingGoalCents?: number;
  goals?: GoalItemInput[];
  donationOptions?: DonationOptionsInput;
  attribution?: AttributionInput;
}

export interface SecurityAuditFundraisePayload {
  initiativeType: 'security_audit';
  name: string;
  description: string;
  industry?: string;
  websiteUrl?: string;
  ciiProjectId?: string;
  cocUrl?: string;
  repositoryUrl?: string;
  logoUrl?: string;
  licenseType?: string;
  currentSecurityStrategy?: string;
  fundingGoalCents?: number;
  primaryContact?: FundraiseContactInput;
  secondaryContact?: FundraiseContactInput;
  technicalLead?: FundraiseContactInput;
  donationOptions?: DonationOptionsInput;
  attribution?: AttributionInput;
}

export interface EventFundraisePayload {
  initiativeType: 'event';
  name: string;
  description: string;
  industry?: string;
  websiteUrl?: string;
  registrationUrl?: string;
  startDate?: string;
  endDate?: string;
  city?: string;
  country?: string;
  isOnline?: boolean;
  logoUrl?: string;
  beneficiaries?: FundraiseBeneficiaryInput[];
  sponsorshipGoalCents?: number;
  budgetDistribution?: GoalItemInput[];
  donationOptions?: DonationOptionsInput;
  attribution?: AttributionInput;
}

export interface GeneralFundFundraisePayload {
  initiativeType: 'general_fund';
  name: string;
  description: string;
  industry?: string;
  websiteUrl?: string;
  logoUrl?: string;
  beneficiaries?: FundraiseBeneficiaryInput[];
  annualFundingGoalCents?: number;
  donationOptions?: DonationOptionsInput;
  attribution?: AttributionInput;
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
