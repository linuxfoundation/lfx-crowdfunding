// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Register @linuxfoundation/lfx-ui-core custom elements (client-only).
// The package self-registers its web components on import — no explicit call needed.
export default defineNuxtPlugin(async () => {
  await import('@linuxfoundation/lfx-ui-core');
});
