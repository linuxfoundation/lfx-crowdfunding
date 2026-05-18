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
  | 'documentation';

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
  logoFileName: string;
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
