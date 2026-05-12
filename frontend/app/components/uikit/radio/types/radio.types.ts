// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT
export const radioSizes = ['small', 'default'] as const;
export type RadioSize = (typeof radioSizes)[number];
