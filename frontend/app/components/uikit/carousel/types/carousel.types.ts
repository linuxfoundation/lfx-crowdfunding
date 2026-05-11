// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

export type CarouselData = Record<string, number | string | boolean | null | undefined>[];

export interface CarouselProps<T> {
  value: T[];
  circular?: boolean;
  showPagination?: boolean;
}
