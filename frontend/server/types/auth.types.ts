// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

export interface ProtectedRoute {
  match: (path: string) => boolean;
  methods?: string[];
}
