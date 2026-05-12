// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT
export const toggleSizes = ['small', 'default'] as const;
export type ToggleSize = (typeof toggleSizes)[number];
