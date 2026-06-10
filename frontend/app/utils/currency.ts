// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

/**
 * Parses a user-entered dollar string (e.g. "$50,000" or "50000") into cents.
 * Returns undefined if the value is empty, zero, or non-numeric.
 */
export function parseDollarsToCents(value: string | undefined): number | undefined {
  if (!value) return undefined;
  const num = parseFloat(value.replace(/[^0-9.]/g, ''));
  return isNaN(num) || num <= 0 ? undefined : Math.round(num * 100);
}

/**
 * Formats a cents amount to a compact dollar string (e.g. "$1.5K", "$2.3M").
 * Rounding is performed in integer space on the cents value to avoid JS
 * floating-point representation errors (e.g. 1005 dollars as "$1.01K" not "$1.00K").
 *
 * @param cents - Amount in cents (integer)
 * @returns Compact dollar string
 */
export function formatAmountCents(cents: number): string {
  // $1M+ — round to nearest 0.1 M
  if (cents >= 100_000_000) {
    const tenthsM = Math.round(cents / 10_000_000);
    return `$${(tenthsM / 10).toFixed(1).replace(/\.0$/, '')}M`;
  }
  // $1K+ — round to nearest 0.01 K (i.e. nearest $10 in cents space)
  if (cents >= 100_000) {
    const hundredthsK = Math.round(cents / 1_000);
    return `$${(hundredthsK / 100).toFixed(2).replace(/\.?0+$/, '')}K`;
  }
  return `$${(cents / 100).toLocaleString()}`;
}
