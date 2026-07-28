// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Re-export the SDK's User type so the rest of the app doesn't import from
// @datadog/browser-rum directly.
export type { User as DatadogRumUser } from '@datadog/browser-rum';

import type { User } from '@datadog/browser-rum';

export interface DatadogIdentifiedUser extends User {
  id: string;
}
