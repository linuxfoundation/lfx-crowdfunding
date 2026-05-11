// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT
export const progressBarTypes = ['normal', 'positive', 'warning', 'negative'] as const;

export type ProgressBarType = (typeof progressBarTypes)[number];
