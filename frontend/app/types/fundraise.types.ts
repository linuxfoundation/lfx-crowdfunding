// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

export type InitiativeType = 'project' | 'security_audit' | 'general_fund' | 'event';

export interface TopicOption {
  value: string;
  label: string;
}

export type ProjectHostingType = 'github' | 'git_url';

export interface GitHubRepo {
  id: number;
  name: string;
  fullName: string;
  description: string;
  stars: number;
}

export interface Beneficiary {
  name: string;
  email: string;
}

export type FundCategory =
  | 'development'
  | 'marketing'
  | 'meetups'
  | 'bug_bounty'
  | 'travel'
  | 'documentation'
  | 'venue'
  | 'food_beverage'
  | 'equipment';

export interface GoalItem {
  category: FundCategory;
  label: string;
  description: string;
  enabled: boolean;
  percentage: number;
}

export interface ProjectDetailsData {
  projectName: string;
  elevatorPitch: string;
  topics: string[];
  repositoryUrl: string;
  websiteUrl: string;
  ciiProjectId: string;
  codeOfConductUrl: string;
  logoUrl: string;
  beneficiaries: Beneficiary[];
  annualFundingGoal: string;
  goals: GoalItem[];
}

export interface ComplianceData {
  ofacConfirmed: boolean;
  termsAccepted: boolean;
}

export type SponsorshipTierName = 'platinum' | 'gold' | 'silver' | 'bronze';

export interface SponsorshipTierConfig {
  name: SponsorshipTierName;
  enabled: boolean;
  goal: string;
  benefits: string[];
}

export type DonationOptionsMode = 'tiers' | 'open';

export interface DonationOptionsData {
  mode: DonationOptionsMode;
  tiers: SponsorshipTierConfig[];
}

export interface ProjectFormData {
  hostingType: ProjectHostingType | null;
  selectedRepo: string | null;
  details: ProjectDetailsData;
  donationOptions: DonationOptionsData;
  compliance: ComplianceData;
}

export interface InitiativeDetailsData {
  name: string;
  elevatorPitch: string;
  topics: string[];
  repositoryUrl?: string;
  websiteUrl: string;
}

export interface ContactPerson {
  firstName: string;
  lastName: string;
  email: string;
  phone: string;
  preferredContact: 'email' | 'phone';
}

export interface SecurityAuditFormData {
  auditName: string;
  elevatorPitch: string;
  topics: string[];
  repositoryUrl: string;
  websiteUrl: string;
  logoUrl: string;
  ciiProjectId: string;
  licenseType: string;
  currentSecurityStrategy: string;
  codeOfConductUrl: string;
  primaryContact: ContactPerson;
  secondaryContact: ContactPerson;
  technicalLead: ContactPerson;
  fundingGoal: string;
  donationOptions: DonationOptionsData;
  compliance: ComplianceData;
}

export interface FundDistributionData {
  goal: string;
  distribution: GoalItem[];
}

export interface EventFormData {
  name: string;
  elevatorPitch: string;
  topics: string[];
  websiteUrl: string;
  registrationUrl: string;
  startDate: string;
  endDate: string;
  city: string;
  country: string;
  logoUrl: string;
  beneficiaries: Beneficiary[];
  sponsorshipGoal: string;
  budgetDistribution: GoalItem[];
  donationOptions: DonationOptionsData;
  compliance: ComplianceData;
}

export interface GeneralFundFormData {
  name: string;
  elevatorPitch: string;
  topics: string[];
  websiteUrl: string;
  logoUrl: string;
  beneficiaries: Beneficiary[];
  annualFundingGoal: string;
  donationOptions: DonationOptionsData;
  compliance: ComplianceData;
}

export interface FundraiseDetailsForm {
  name: string;
  description: string;
  githubUrl: string;
  tags: string;
  auditScope: string;
  eventDate: string;
  location: string;
  eventbriteUrl: string;
}
