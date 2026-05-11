// Copyright (c) 2025 The Linux Foundation and each contributor.
// SPDX-License-Identifier: MIT

// Register @linuxfoundation/lfx-ui-core custom elements (client-only).
// The package self-registers its web components on import — no explicit call needed.
export default defineNuxtPlugin(async () => {
  await import('@linuxfoundation/lfx-ui-core');
});
