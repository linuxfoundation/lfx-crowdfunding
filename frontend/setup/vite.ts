// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Web components exported by @linuxfoundation/lfx-ui-core
const LFX_WEB_COMPONENTS = new Set(['lfx-footer']);

export default {
  // Keep Vue's hydration mismatch details in prod builds so RUM shows which node differed.
  define: { __VUE_PROD_HYDRATION_MISMATCH_DETAILS__: 'true' },
  vue: {
    template: {
      compilerOptions: {
        isCustomElement: (tag: string) => LFX_WEB_COMPONENTS.has(tag),
      },
    },
  },
};
