// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT
export interface Summary {
  current: number;
  previous: number;
  percentageChange: number | undefined;
  changeValue: number;
  periodFrom: string;
  periodTo: string;
  maintainerCount?: number;
}

export interface Meta {
  offset: number;
  limit: number;
  total: number;
}
