// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT
export const chipSizes = ['xsmall', 'small', 'default'] as const;
export const chipTypes = ['bordered', 'default'] as const;
export type ChipSize = (typeof chipSizes)[number];
export type ChipType = (typeof chipTypes)[number];
