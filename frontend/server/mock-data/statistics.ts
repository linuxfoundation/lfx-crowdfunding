// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type {
  StatisticsOverview,
  FundingCategory,
  DonorBreakdown,
  MonthlyDonations,
  TopDonorsResponse,
} from '#shared/types/statistics.types';
import type { RecentDonation } from '#shared/types/initiative-detail.types';

export const MOCK_OVERVIEW: StatisticsOverview = {
  totalRaisedCents: 580_000_000,
  supporterCount: 1842,
  annualGoalCents: 1_000_000_000,
  activeInitiatives: 47,
};

export const MOCK_FUNDING_BY_CATEGORY: FundingCategory[] = [
  {
    id: 'security',
    name: 'Security',
    icon: 'shield-check',
    raisedCents: 175_000_000,
    goalCents: 250_000_000,
    supporterCount: 620,
  },
  {
    id: 'infrastructure',
    name: 'Infrastructure',
    icon: 'gear',
    raisedCents: 148_000_000,
    goalCents: 250_000_000,
    supporterCount: 540,
  },
  {
    id: 'community',
    name: 'Community',
    icon: 'people-group',
    raisedCents: 112_000_000,
    goalCents: 200_000_000,
    supporterCount: 410,
  },
  {
    id: 'events',
    name: 'Events',
    icon: 'calendar',
    raisedCents: 89_000_000,
    goalCents: 200_000_000,
    supporterCount: 310,
  },
  {
    id: 'travel',
    name: 'Travel',
    icon: 'plane',
    raisedCents: 56_000_000,
    goalCents: 100_000_000,
    supporterCount: 188,
  },
];

export const MOCK_DONOR_BREAKDOWN: DonorBreakdown = {
  avgDonationCents: 320_000,
  organizationsCents: 362_000_000,
  individualsCents: 218_000_000,
};

export const MOCK_MONTHLY_DONATIONS: MonthlyDonations = {
  buckets: [
    { year: 2024, month: 6, totalCents: 32_000_000, supporters: 89, newSupporters: 12 },
    { year: 2024, month: 7, totalCents: 35_500_000, supporters: 97, newSupporters: 14 },
    { year: 2024, month: 8, totalCents: 28_000_000, supporters: 74, newSupporters: 8 },
    { year: 2024, month: 9, totalCents: 41_000_000, supporters: 112, newSupporters: 18 },
    { year: 2024, month: 10, totalCents: 38_500_000, supporters: 105, newSupporters: 11 },
    { year: 2024, month: 11, totalCents: 45_000_000, supporters: 118, newSupporters: 16 },
    { year: 2024, month: 12, totalCents: 52_000_000, supporters: 143, newSupporters: 22 },
    { year: 2025, month: 1, totalCents: 39_000_000, supporters: 101, newSupporters: 9 },
    { year: 2025, month: 2, totalCents: 43_500_000, supporters: 115, newSupporters: 13 },
    { year: 2025, month: 3, totalCents: 47_000_000, supporters: 128, newSupporters: 17 },
    { year: 2025, month: 4, totalCents: 48_200_000, supporters: 124, newSupporters: 15 },
    { year: 2025, month: 5, totalCents: 55_000_000, supporters: 151, newSupporters: 24 },
  ],
};

export const MOCK_TOP_DONORS: TopDonorsResponse = {
  organizations: [
    { rank: 1, id: 'nimbus-networks', name: 'Nimbus Networks', amountCents: 7_500_000 },
    { rank: 2, id: 'skytech-solutions', name: 'SkyTech Solutions', amountCents: 7_500_000 },
    {
      rank: 3,
      id: 'cloudsphere-innovations',
      name: 'CloudSphere Innovations',
      amountCents: 7_500_000,
    },
    { rank: 4, id: 'aether-systems', name: 'Aether Systems', amountCents: 7_500_000 },
    {
      rank: 5,
      id: 'stratosphere-technologies',
      name: 'Stratosphere Technologies',
      amountCents: 7_500_000,
    },
    { rank: 6, id: 'cumulus-computing', name: 'Cumulus Computing', amountCents: 7_500_000 },
    {
      rank: 7,
      id: 'vertex-cloud-services',
      name: 'Vertex Cloud Services',
      amountCents: 7_500_000,
    },
    { rank: 8, id: 'quantum-cloudworks', name: 'Quantum Cloudworks', amountCents: 7_500_000 },
    { rank: 9, id: 'nebula-networks', name: 'Nebula Networks', amountCents: 7_500_000 },
    {
      rank: 10,
      id: 'horizon-cloud-solutions',
      name: 'Horizon Cloud Solutions',
      amountCents: 7_500_000,
    },
  ],
  individuals: [
    { rank: 1, id: 'leslie-alexander', name: 'Leslie Alexander', amountCents: 7_500_000 },
    { rank: 2, id: 'devon-lane', name: 'Devon Lane', amountCents: 7_500_000 },
    { rank: 3, id: 'robert-fox', name: 'Robert Fox', amountCents: 7_500_000 },
    { rank: 4, id: 'savannah-nguyen', name: 'Savannah Nguyen', amountCents: 7_500_000 },
    { rank: 5, id: 'esther-howard', name: 'Esther Howard', amountCents: 7_500_000 },
    { rank: 6, id: 'brooklyn-simmons', name: 'Brooklyn Simmons', amountCents: 7_500_000 },
    { rank: 7, id: 'annette-black', name: 'Annette Black', amountCents: 7_500_000 },
    { rank: 8, id: 'floyd-miles', name: 'Floyd Miles', amountCents: 7_500_000 },
    { rank: 9, id: 'kathryn-murphy', name: 'Kathryn Murphy', amountCents: 7_500_000 },
    { rank: 10, id: 'bessie-cooper', name: 'Bessie Cooper', amountCents: 7_500_000 },
  ],
};

export const MOCK_RECENT_DONATIONS: RecentDonation[] = [
  {
    id: '1',
    donorName: 'Google Cloud',
    donorType: 'organization',
    amountCents: 10_000_000,
    timeAgo: '2h ago',
    initiativeId: 'kubernetes-security-audit',
    initiativeName: 'Kubernetes Security Audit',
  },
  {
    id: '2',
    donorName: 'Priya Sharma',
    donorType: 'member',
    amountCents: 25_000,
    timeAgo: '2h ago',
    initiativeId: 'kubernetes-security-audit',
    initiativeName: 'Kubernetes Security Audit',
  },
  {
    id: '3',
    donorName: 'Cisco',
    donorType: 'organization',
    amountCents: 10_000_000,
    timeAgo: '3h ago',
    initiativeId: 'linux-kernel-release-signing',
    initiativeName: 'Linux Kernel Release Signing',
  },
  {
    id: '4',
    donorName: 'Alex Petrov',
    donorType: 'member',
    amountCents: 1_500_000,
    timeAgo: '5h ago',
    initiativeId: 'thanos',
    initiativeName: 'Thanos',
  },
  {
    id: '5',
    donorName: 'Lena Mueller',
    donorType: 'member',
    amountCents: 5_000,
    timeAgo: '6h ago',
    initiativeId: 'linux-kernel-bug-fixing-2022',
    initiativeName: 'Linux Kernel Bug Fixing',
  },
  {
    id: '6',
    donorName: 'Red Hat',
    donorType: 'organization',
    amountCents: 5_000_000,
    timeAgo: '8h ago',
    initiativeId: 'linux-kernel-vulnerability-remediation',
    initiativeName: 'Linux Kernel Vulnerability Remediation',
  },
  {
    id: '7',
    donorName: 'Tom Wilson',
    donorType: 'member',
    amountCents: 100_000,
    timeAgo: '10h ago',
    initiativeId: 'accordproject',
    initiativeName: 'Accord Project',
  },
  {
    id: '8',
    donorName: 'AWS',
    donorType: 'organization',
    amountCents: 7_500_000,
    timeAgo: '12h ago',
    initiativeId: 'cloudnativehacks',
    initiativeName: 'CloudNativeHacks',
  },
];
