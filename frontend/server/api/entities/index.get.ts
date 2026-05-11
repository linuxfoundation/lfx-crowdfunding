// Copyright (c) 2025 The Linux Foundation and each contributor.
// SPDX-License-Identifier: MIT

export default defineEventHandler(async () => {
  // TODO: replace with real DB queries once the Go backend / Postgres is wired up
  const entities = [
    {
      id: '1',
      entityId: 'kubernetes',
      ownerId: 'lf',
      name: 'Kubernetes',
      description: 'Production-Grade Container Scheduling and Management.',
      status: 'active',
      entityType: 'project',
      color: '#326CE5',
      createdOn: new Date('2024-01-01'),
      updatedOn: new Date('2025-01-01'),
      fundingStatus: {
        totalAnnualGoalInCents: 10_000_000,
        totalDonationsInCents: 8_245_000,
      },
    },
    {
      id: '2',
      entityId: 'lf-mentorship-2025',
      ownerId: 'lf',
      name: 'LF Mentorship 2025',
      description: 'Pair aspiring open source contributors with experienced mentors.',
      status: 'active',
      entityType: 'mentorship',
      color: '#4CAF50',
      createdOn: new Date('2025-01-01'),
      updatedOn: new Date('2025-01-01'),
      fundingStatus: {
        totalAnnualGoalInCents: 5_000_000,
        totalDonationsInCents: 2_310_000,
      },
    },
    {
      id: '3',
      entityId: 'open-source-summit-2025',
      ownerId: 'lf',
      name: 'Open Source Summit 2025',
      description:
        'The premier event for open source developers, technologists, and community leaders.',
      status: 'active',
      entityType: 'event',
      color: '#FF9800',
      createdOn: new Date('2025-01-01'),
      updatedOn: new Date('2025-01-01'),
      eventStartDate: '2025-09-15',
      eventEndDate: '2025-09-18',
      fundingStatus: {
        totalAnnualGoalInCents: 7_500_000,
        totalDonationsInCents: 1_200_000,
      },
    },
  ];

  return {
    data: entities,
    total: entities.length,
  };
});
