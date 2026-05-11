// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT
export const popoverTrigger = ['click', 'hover'] as const;

export type PopoverTrigger = (typeof popoverTrigger)[number];
