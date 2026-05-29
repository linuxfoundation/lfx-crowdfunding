// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

export interface PresignedURLWire {
  upload_url: string;
  destination_url: string;
  required_headers: Record<string, string>;
}
