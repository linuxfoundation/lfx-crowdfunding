// Copyright (c) 2025 The Linux Foundation and each contributor.
// SPDX-License-Identifier: MIT

// Web components exported by @linuxfoundation/lfx-ui-core
const LFX_WEB_COMPONENTS = new Set(['lfx-footer']);

export default {
  vue: {
    template: {
      compilerOptions: {
        isCustomElement: (tag: string) => LFX_WEB_COMPONENTS.has(tag),
      },
    },
  },
};
