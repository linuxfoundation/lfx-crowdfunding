// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Shape of an ofetch FetchError from a BFF route (h3 createError), or a plain
// Error. Fields are unknown: Nitro's error body sets `error: true`.
export interface PaymentErrorLike {
  message?: unknown;
  data?: { error?: unknown; message?: unknown; statusMessage?: unknown };
}
