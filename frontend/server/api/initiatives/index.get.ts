// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

export default defineEventHandler(async () => {
  // TODO: replace with a proxy call to the Go backend API once wired up.
  // In our architecture Nuxt's server layer owns auth only — business data comes
  // from the Go microservice. This handler returns fixture data for scaffolding.
  const initiatives = [
    {
      id: '1',
      initiativeId: 'kubernetes',
      ownerId: 'lf',
      name: 'Kubernetes',
      description: 'Production-Grade Container Scheduling and Management.',
      status: 'active',
      initiativeType: 'project',
      color: '#326CE5',
      createdOn: '2024-01-01T00:00:00.000Z',
      updatedOn: '2025-01-01T00:00:00.000Z',
      fundingStatus: {
        totalAnnualGoalInCents: 10_000_000,
        amountRaisedCents: 8_245_000,
      },
    },
    {
      id: '2',
      initiativeId: 'lf-mentorship-2025',
      ownerId: 'lf',
      name: 'LF Mentorship 2025',
      description: 'Pair aspiring open source contributors with experienced mentors.',
      status: 'active',
      initiativeType: 'mentorship',
      color: '#4CAF50',
      createdOn: '2025-01-01T00:00:00.000Z',
      updatedOn: '2025-01-01T00:00:00.000Z',
      fundingStatus: {
        totalAnnualGoalInCents: 5_000_000,
        amountRaisedCents: 2_310_000,
      },
    },
    {
      id: '3',
      initiativeId: 'open-source-summit-2025',
      ownerId: 'lf',
      name: 'Open Source Summit 2025',
      description:
        'The premier event for open source developers, technologists, and community leaders.',
      status: 'active',
      initiativeType: 'event',
      color: '#FF9800',
      createdOn: '2025-01-01T00:00:00.000Z',
      updatedOn: '2025-01-01T00:00:00.000Z',
      eventStartDate: '2025-09-15',
      eventEndDate: '2025-09-18',
      fundingStatus: {
        totalAnnualGoalInCents: 7_500_000,
        amountRaisedCents: 1_200_000,
      },
    },
  ];

  return {
    data: initiatives,
    total: initiatives.length,
  };
});
