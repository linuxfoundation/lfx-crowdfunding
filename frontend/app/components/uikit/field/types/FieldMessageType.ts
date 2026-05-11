// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT
export const fieldMessageTypes = ['error', 'warning', 'hint', 'info'] as const;

export type FieldMessageType = (typeof fieldMessageTypes)[number];
