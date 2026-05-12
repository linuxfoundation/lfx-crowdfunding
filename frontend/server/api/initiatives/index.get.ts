// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { InitiativeBase, InitiativesResponse } from '#shared/types/initiative.types';

const ALL_INITIATIVES: InitiativeBase[] = [
  {
    id: '1',
    initiativeId: 'kubernetes-security-audit',
    ownerId: 'lf',
    name: 'Kubernetes Security Audit',
    description:
      "Comprehensive security audit for the world's most popular container orchestrator.",
    status: 'active',
    initiativeType: 'security_audit',
    industry: 'Security, Cloud Native, CNCF',
    color: '#E05C00',
    createdOn: '2024-01-01T00:00:00.000Z',
    updatedOn: '2025-05-10T00:00:00.000Z',
    initiativeStats: { backers: 400, sponsors: 12, totalRaised: 200_000 },
    fundingStatus: { totalAnnualGoalInCents: 40_000_000, amountRaisedCents: 26_000_000 },
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
  },
  {
    id: '3',
    initiativeId: 'linux-kernel-release-signing',
    ownerId: 'lf',
    name: 'Linux Kernel Release Signing Security Assessment',
    description:
      'The Linux Foundation worked with OSTIF to facilitate an in-depth security assessment of the kernel release signing process.',
    status: 'active',
    initiativeType: 'security_audit',
    industry: 'Security, Linux, Kernel',
    color: '#E05C00',
    createdOn: '2024-03-01T00:00:00.000Z',
    updatedOn: '2025-05-05T00:00:00.000Z',
    initiativeStats: { backers: 400, sponsors: 10, totalRaised: 200_000 },
    fundingStatus: { totalAnnualGoalInCents: 40_000_000, amountRaisedCents: 26_000_000 },
  },
  {
    id: '4',
    initiativeId: 'kubernetes-austin-meetup',
    ownerId: 'lf',
    name: 'Kubernetes Austin Meetup',
    description:
      "Comprehensive security audit for the world's most popular container orchestrator.",
    status: 'active',
    initiativeType: 'event',
    industry: 'Cloud Native, Kubernetes, Meetup',
    color: '#FF9800',
    createdOn: '2025-01-15T00:00:00.000Z',
    updatedOn: '2025-05-03T00:00:00.000Z',
    eventStartDate: '2025-07-20',
    eventEndDate: '2025-07-20',
    initiativeStats: { backers: 400, sponsors: 5, totalRaised: 200_000 },
    fundingStatus: { totalAnnualGoalInCents: 40_000_000, amountRaisedCents: 26_000_000 },
  },
  {
    id: '5',
    initiativeId: 'accordproject',
    ownerId: 'lf',
    name: 'accordproject',
    description:
      'The Accord Project is an open ecosystem enabling anyone to build smart agreements and documents.',
    status: 'active',
    initiativeType: 'project',
    industry: 'Legal Tech, Smart Contracts, Open Source',
    color: '#326CE5',
    createdOn: '2023-09-01T00:00:00.000Z',
    updatedOn: '2025-04-28T00:00:00.000Z',
    initiativeStats: { backers: 400, sponsors: 7, totalRaised: 200_000 },
    fundingStatus: { totalAnnualGoalInCents: 40_000_000, amountRaisedCents: 26_000_000 },
  },
  {
    id: '6',
    initiativeId: 'cloudnativehacks',
    ownerId: 'lf',
    name: 'CloudNativeHacks',
    description:
      'Provide prize money to the winners of CloudNativeHacks, a cloud native hackathon event.',
    status: 'active',
    initiativeType: 'general_fund',
    industry: 'Hackathon, Cloud Native, CNCF',
    color: '#9C27B0',
    createdOn: '2025-02-01T00:00:00.000Z',
    updatedOn: '2025-04-20T00:00:00.000Z',
    initiativeStats: { backers: 400, sponsors: 9, totalRaised: 380_000 },
    fundingStatus: { totalAnnualGoalInCents: 40_000_000, amountRaisedCents: 36_000_000 },
  },
  {
    id: '7',
    initiativeId: 'linux-kernel-vulnerability-remediation',
    ownerId: 'lf',
    name: 'Linux Kernel Vulnerability Remediation and Reporting Security Review',
    description:
      'The Linux Kernel Vulnerability Reporting and Remediation Review covered practices and policies around how security vulnerabilities are reported, processed, and disclosed.',
    status: 'active',
    initiativeType: 'security_audit',
    industry: 'Security, Linux, Kernel',
    color: '#E05C00',
    createdOn: '2024-02-01T00:00:00.000Z',
    updatedOn: '2025-04-15T00:00:00.000Z',
    initiativeStats: { backers: 400, sponsors: 11, totalRaised: 200_000 },
    fundingStatus: { totalAnnualGoalInCents: 40_000_000, amountRaisedCents: 26_000_000 },
  },
  {
    id: '8',
    initiativeId: 'ml-predict-deforestation',
    ownerId: 'lf',
    name: 'Using Machine Learning to Predict Deforestation',
    description:
      'Climate change is the biggest challenge of our time. We need all the help we can get to reduce deforestation using machine learning.',
    status: 'active',
    initiativeType: 'mentorship',
    industry: 'Machine Learning, Climate, Open Source',
    color: '#4CAF50',
    createdOn: '2024-07-01T00:00:00.000Z',
    updatedOn: '2025-04-10T00:00:00.000Z',
    initiativeStats: { backers: 400, sponsors: 6, totalRaised: 200_000 },
    fundingStatus: { totalAnnualGoalInCents: 40_000_000, amountRaisedCents: 26_000_000 },
  },
  {
    id: '9',
    initiativeId: 'linux-kernel-bug-fixing-2022',
    ownerId: 'lf',
    name: 'Linux Kernel Bug Fixing Spring 2022',
    description:
      'There are many bugs being found in the Linux kernel by automated tools, yet not many people are working to fix them.',
    status: 'active',
    initiativeType: 'mentorship',
    industry: 'Linux, Kernel, Bug Fixing',
    color: '#4CAF50',
    createdOn: '2022-03-01T00:00:00.000Z',
    updatedOn: '2025-04-01T00:00:00.000Z',
    initiativeStats: { backers: 400, sponsors: 4, totalRaised: 200_000 },
    fundingStatus: { totalAnnualGoalInCents: 40_000_000, amountRaisedCents: 26_000_000 },
  },
];

export default defineEventHandler(async (event): Promise<InitiativesResponse> => {
  // TODO: replace with a proxy call to the Go backend API once wired up.
  // In our architecture Nuxt's server layer owns auth only — business data comes
  // from the Go microservice. This handler returns fixture data for scaffolding.
  const { search, type, sort } = getQuery(event);

  const query = typeof search === 'string' ? search.trim().toLowerCase() : '';
  const typeFilter = typeof type === 'string' && type !== 'all' ? type : '';
  const sortBy = typeof sort === 'string' ? sort : 'recent';

  let result = ALL_INITIATIVES.filter((i) => {
    const matchesSearch =
      !query || i.name.toLowerCase().includes(query) || i.description.toLowerCase().includes(query);
    const matchesType = !typeFilter || i.initiativeType === typeFilter;
    return matchesSearch && matchesType;
  });

  if (sortBy === 'name') {
    result = [...result].sort((a, b) => a.name.localeCompare(b.name));
  } else if (sortBy === 'funded') {
    result = [...result].sort((a, b) => {
      const pctA =
        (a.fundingStatus?.amountRaisedCents ?? 0) / (a.fundingStatus?.totalAnnualGoalInCents ?? 1);
      const pctB =
        (b.fundingStatus?.amountRaisedCents ?? 0) / (b.fundingStatus?.totalAnnualGoalInCents ?? 1);
      return pctB - pctA;
    });
  } else {
    result = [...result].sort(
      (a, b) => new Date(b.updatedOn).getTime() - new Date(a.updatedOn).getTime(),
    );
  }

  return { data: result, total: result.length };
});
