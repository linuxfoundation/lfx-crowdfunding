// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

export interface PresignedURLResult {
  uploadUrl: string;
  destinationUrl: string;
  requiredHeaders: Record<string, string>;
}

export type AllowedLogoMimeType = 'image/png' | 'image/jpeg' | 'image/gif' | 'image/webp';

export const ALLOWED_LOGO_MIME_TYPES: AllowedLogoMimeType[] = [
  'image/png',
  'image/jpeg',
  'image/gif',
  'image/webp',
];

/** Maximum logo file size accepted (2 MB). Enforced client-side only — S3 does not enforce this. */
export const MAX_LOGO_SIZE_BYTES = 2 * 1024 * 1024;
