// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

export type InitiativeType = 'project' | 'security_audit' | 'general_fund' | 'event';

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

export interface FundDistributionItem {
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
  websiteUrl: string;
  ciiProjectId: string;
  codeOfConductUrl: string;
  logoUrl: string;
  beneficiaries: Beneficiary[];
  annualFundingGoal: string;
  fundDistribution: FundDistributionItem[];
}

export interface ComplianceData {
  ofacConfirmed: boolean;
  termsAccepted: boolean;
}

export interface ProjectFormData {
  hostingType: ProjectHostingType | null;
  selectedRepo: string | null;
  details: ProjectDetailsData;
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
  compliance: ComplianceData;
}

export interface FundDistributionData {
  goal: string;
  distribution: FundDistributionItem[];
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
  budgetDistribution: FundDistributionItem[];
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
