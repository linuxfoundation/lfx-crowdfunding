// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type {
  DonationOptionsData,
  DonationOptionsMode,
  SponsorshipTierName,
} from '~/types/fundraise.types';

export interface DonationModeOption {
  value: DonationOptionsMode;
  label: string;
  description: string;
}

export const DONATION_MODE_OPTIONS: DonationModeOption[] = [
  {
    value: 'tiers',
    label: 'Define donation tiers',
    description: 'Offer set amounts like Gold or Bronze, each with its own benefits.',
  },
  {
    value: 'open',
    label: 'Skip tiers',
    description: 'Let sponsors enter any amount, with no predefined levels.',
  },
];

export const SPONSORSHIP_TIER_NAMES: SponsorshipTierName[] = [
  'platinum',
  'gold',
  'silver',
  'bronze',
];

export const SPONSORSHIP_TIER_LABEL: Record<SponsorshipTierName, string> = {
  platinum: 'Platinum',
  gold: 'Gold',
  silver: 'Silver',
  bronze: 'Bronze',
};

// Radial-gradient "orb" look from the Figma tier icons: a soft highlight top-left, a soft
// shadow bottom-right, over the tier's base color. Applied as the background of a small
// clipped shape (diamond for platinum, circle for the rest) — see fundraise-donation-tier-card.vue.
export const SPONSORSHIP_TIER_ICON_GRADIENT: Record<SponsorshipTierName, string> = {
  platinum:
    'radial-gradient(circle at 30% 25%, rgba(255, 255, 255, 0.6), rgba(255, 255, 255, 0) 70%), radial-gradient(circle at 75% 80%, rgba(0, 0, 0, 0.25), rgba(0, 0, 0, 0) 75%), #bebebe',
  gold: 'radial-gradient(circle at 30% 25%, rgba(255, 255, 255, 0.6), rgba(255, 255, 255, 0) 70%), radial-gradient(circle at 75% 80%, rgba(0, 0, 0, 0.25), rgba(0, 0, 0, 0) 75%), #d7a262',
  silver:
    'radial-gradient(circle at 30% 25%, rgba(255, 255, 255, 0.6), rgba(255, 255, 255, 0) 70%), radial-gradient(circle at 75% 80%, rgba(0, 0, 0, 0.25), rgba(0, 0, 0, 0) 75%), #9e9fa1',
  bronze:
    'radial-gradient(circle at 30% 25%, rgba(255, 255, 255, 0.6), rgba(255, 255, 255, 0) 70%), radial-gradient(circle at 75% 80%, rgba(0, 0, 0, 0.25), rgba(0, 0, 0, 0) 75%), #b97a50',
};

export const createDefaultDonationOptions = (): DonationOptionsData => ({
  mode: 'tiers',
  tiers: SPONSORSHIP_TIER_NAMES.map((name) => ({ name, enabled: false, goal: '', benefits: [] })),
});
