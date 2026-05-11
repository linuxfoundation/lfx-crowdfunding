// Copyright (c) 2025 The Linux Foundation and each contributor.
// SPDX-License-Identifier: MIT

export default defineEventHandler(async () => {
  // TODO: replace with real DB queries once the Go backend / Postgres is wired up
  const campaigns = [
    {
      id: '1',
      slug: 'kubernetes',
      title: 'Kubernetes',
      description: 'Production-Grade Container Scheduling and Management.',
      type: 'project',
      goalAmount: 100_000,
      raisedAmount: 82_450,
      currency: '$',
    },
    {
      id: '2',
      slug: 'lf-mentorship-2025',
      title: 'LF Mentorship 2025',
      description: 'Pair aspiring open source contributors with experienced mentors.',
      type: 'mentorship',
      goalAmount: 50_000,
      raisedAmount: 23_100,
      currency: '$',
    },
    {
      id: '3',
      slug: 'open-source-summit',
      title: 'Open Source Summit 2025',
      description:
        'The premier event for open source developers, technologists, and community leaders.',
      type: 'event',
      goalAmount: 75_000,
      raisedAmount: 12_000,
      currency: '$',
    },
  ];

  return {
    data: campaigns,
    total: campaigns.length,
  };
});
