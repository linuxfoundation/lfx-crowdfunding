// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT
export interface Pagination<N> {
  page: number;
  pageSize: number;
  total: number;
  data: N[];
}
