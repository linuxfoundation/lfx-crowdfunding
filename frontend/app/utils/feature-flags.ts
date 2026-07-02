// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// ponytail: sponsorship tiers aren't launched yet, hide behind non-production envs until GA.
export const isSponsorshipTiersEnabled = () => useRuntimeConfig().public.appEnv !== 'production';
