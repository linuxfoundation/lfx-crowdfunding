// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT
declare module '*.vue' {
  import type { DefineComponent } from 'vue';
  const component: DefineComponent<object, object, unknown>;
  export default component;
}
