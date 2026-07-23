// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// The actual per-environment disallow decision is made at request time in
// server/plugins/robots.ts, since this file is evaluated at build time and
// the same built image is deployed to multiple environments (LFXV2-2700).
export default {};
