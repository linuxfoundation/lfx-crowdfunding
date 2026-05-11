// Copyright (c) 2025 The Linux Foundation and each contributor.
// SPDX-License-Identifier: MIT
import { useQuery } from '@tanstack/vue-query';

export interface Campaign {
  id: string;
  slug: string;
  title: string;
  description: string;
  type: 'project' | 'mentorship' | 'general_fund' | 'event';
  goalAmount: number;
  raisedAmount: number;
  currency: string;
}

export interface CampaignsResponse {
  data: Campaign[];
  total: number;
}

export function useCampaigns() {
  return useQuery<CampaignsResponse>({
    queryKey: ['campaigns'],
    queryFn: () => $fetch<CampaignsResponse>('/api/campaigns'),
  });
}
