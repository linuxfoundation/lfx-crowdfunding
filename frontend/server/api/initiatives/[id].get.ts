// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { InitiativeDetail } from '#shared/types/initiative-detail.types';

const INITIATIVES: InitiativeDetail[] = [
  {
    id: '1',
    initiativeId: 'kubernetes-security-audit',
    ownerId: 'lf',
    name: 'Kubernetes Security Audit',
    description:
      "Comprehensive security audit for the world's most popular container orchestrator.",
    status: 'active',
    initiativeType: 'project',
    industry: 'Security, Cloud Native, CNCF',
    color: '#326CE5',
    createdOn: '2024-01-01T00:00:00.000Z',
    updatedOn: '2025-05-10T00:00:00.000Z',
    initiativeStats: { backers: 400, sponsors: 12, totalRaised: 200_000 },
    fundingStatus: { totalAnnualGoalInCents: 40_000_000, amountRaisedCents: 15_600_000 },
    websiteURL: 'https://kubernetes.io',
    githubURL: 'https://github.com/kubernetes/kubernetes',
    currentBalanceCents: 7_000_000,
    sponsors: [
      { name: 'Google Cloud' },
      { name: 'Microsoft' },
      { name: 'Amazon' },
      { name: 'Red Hat' },
      { name: 'VMware' },
      { name: 'Cisco' },
      { name: 'IBM' },
    ],
    recentDonations: [
      {
        id: '1',
        donorName: 'Google Cloud',
        donorType: 'organization',
        amountCents: 7_500_000,
        timeAgo: '2h ago',
      },
      {
        id: '2',
        donorName: 'Priya Sharma',
        donorType: 'member',
        amountCents: 25_000,
        timeAgo: '2h ago',
      },
      {
        id: '3',
        donorName: 'Cisco',
        donorType: 'organization',
        amountCents: 10_000_000,
        timeAgo: '3h ago',
      },
      {
        id: '4',
        donorName: 'Alex Petrov',
        donorType: 'member',
        amountCents: 1_500_000,
        timeAgo: '5h ago',
      },
      {
        id: '5',
        donorName: 'Lena Mueller',
        donorType: 'member',
        amountCents: 5_000,
        timeAgo: '6h ago',
      },
    ],
    impactStats: [
      { value: '23', label: 'Websites Secured' },
      { value: '7', label: 'Certificates Issued' },
      { value: '1.2M lines', label: 'Uptime' },
    ],
    projectHealthRating: 'Excellent',
    projectHealthStats: [
      { icon: 'people-group', label: 'Contributors', value: '0.2K' },
      { icon: 'dollar-sign', label: 'Software value', value: '$4.2M' },
      { icon: 'sack-dollar', label: 'Org. dependency', value: '8%' },
      { icon: 'star', label: 'Stars', value: '5K' },
    ],
    fundingGoals: [
      {
        id: '1',
        name: 'Mentee Stipends',
        donatedCents: 7_200_000,
        spentCents: 7_200_000,
        goalCents: 40_000_000,
      },
      {
        id: '2',
        name: 'Mentor Honorariums',
        donatedCents: 7_200_000,
        spentCents: 7_200_000,
        goalCents: 40_000_000,
      },
      {
        id: '3',
        name: 'Events & Graduation',
        donatedCents: 7_200_000,
        spentCents: 7_200_000,
        goalCents: 40_000_000,
      },
      {
        id: '4',
        name: 'Program Operations',
        donatedCents: 7_200_000,
        spentCents: 7_200_000,
        goalCents: 40_000_000,
      },
    ],
  },
  {
    id: '2',
    initiativeId: 'thanos',
    ownerId: 'lf',
    name: 'Thanos',
    description:
      'Thanos is a set of components that can be composed into a highly available metric system with unlimited storage capacity.',
    status: 'active',
    initiativeType: 'mentorship',
    industry: 'Observability, Prometheus, Cloud Native',
    color: '#4CAF50',
    createdOn: '2024-06-01T00:00:00.000Z',
    updatedOn: '2025-05-08T00:00:00.000Z',
    initiativeStats: { backers: 400, sponsors: 8, totalRaised: 200_000 },
    fundingStatus: { totalAnnualGoalInCents: 40_000_000, amountRaisedCents: 26_000_000 },
    currentBalanceCents: 5_200_000,
    sponsors: [{ name: 'Grafana Labs' }, { name: 'Red Hat' }, { name: 'Coralogix' }],
    recentDonations: [
      {
        id: '1',
        donorName: 'Grafana Labs',
        donorType: 'organization',
        amountCents: 5_000_000,
        timeAgo: '1h ago',
      },
      {
        id: '2',
        donorName: 'Jan Novak',
        donorType: 'member',
        amountCents: 10_000,
        timeAgo: '3h ago',
      },
    ],
    impactStats: [
      { value: '15', label: 'Mentees Graduated' },
      { value: '8', label: 'Mentors Active' },
      { value: '3K+', label: 'PRs Merged' },
    ],
    projectHealthStats: [
      { icon: 'users', label: 'Contributors', value: '0.8K' },
      { icon: 'sack-dollar', label: 'Software value', value: '$2.1M' },
      { icon: 'building', label: 'Org. dependency', value: '5%' },
      { icon: 'star', label: 'Stars', value: '12K' },
    ],
    fundingGoals: [
      {
        id: '1',
        name: 'Mentee Stipends',
        donatedCents: 13_000_000,
        spentCents: 10_000_000,
        goalCents: 20_000_000,
      },
      {
        id: '2',
        name: 'Mentor Honorariums',
        donatedCents: 6_000_000,
        spentCents: 5_000_000,
        goalCents: 10_000_000,
      },
      {
        id: '3',
        name: 'Events & Graduation',
        donatedCents: 4_000_000,
        spentCents: 3_000_000,
        goalCents: 5_000_000,
      },
      {
        id: '4',
        name: 'Program Operations',
        donatedCents: 3_000_000,
        spentCents: 2_000_000,
        goalCents: 5_000_000,
      },
    ],
  },
];

function buildFallback(id: string): InitiativeDetail {
  return {
    id,
    initiativeId: id,
    ownerId: 'lf',
    name: id
      .split('-')
      .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
      .join(' '),
    description: 'Open source initiative supported by the Linux Foundation.',
    status: 'active',
    initiativeType: 'project',
    color: '#326CE5',
    createdOn: '2024-01-01T00:00:00.000Z',
    updatedOn: '2025-05-01T00:00:00.000Z',
    initiativeStats: { backers: 100, sponsors: 5, totalRaised: 50_000 },
    fundingStatus: { totalAnnualGoalInCents: 10_000_000, amountRaisedCents: 4_000_000 },
    currentBalanceCents: 2_000_000,
    sponsors: [{ name: 'Linux Foundation' }],
    recentDonations: [],
    impactStats: [],
    projectHealthStats: [],
    fundingGoals: [],
  };
}

export default defineEventHandler(async (event) => {
  const id = getRouterParam(event, 'id');

  if (!id) {
    throw createError({ statusCode: 400, message: 'Missing initiative id' });
  }

  const found = INITIATIVES.find((i) => i.initiativeId === id || i.id === id);
  return found ?? buildFallback(id);
});
