// Copyright (c) 2025 The Linux Foundation and each contributor.
// SPDX-License-Identifier: MIT
export const lfxFontSizes = {
  // Base scale
  xxxs: ['0.5rem', '1.25'] as [string, string],
  xxs: ['0.625rem', '1.25'] as [string, string],
  '2xs': '0.625rem',
  xs: '0.75rem',
  sm: '0.875rem',
  base: '1rem',
  lg: '1.125rem',
  xl: '1.25rem',
  '2xl': '1.5rem',
  '3xl': '1.875rem',
  '4xl': '2.25rem',
  '5xl': '3rem',
  '6xl': '3.75rem',

  // Semantic scale (maps to CSS variables defined in _typography.scss)
  'data-display-1': [
    'var(--lfx-text-data-display-1-font-size)',
    'var(--lfx-text-data-display-1-line-height)',
  ] as [string, string],
  'data-display-2': [
    'var(--lfx-text-data-display-2-font-size)',
    'var(--lfx-text-data-display-2-line-height)',
  ] as [string, string],
  'heading-1': ['var(--lfx-text-heading-1-font-size)', 'var(--lfx-text-heading-1-line-height)'] as [
    string,
    string,
  ],
  'heading-2': ['var(--lfx-text-heading-2-font-size)', 'var(--lfx-text-heading-2-line-height)'] as [
    string,
    string,
  ],
  'heading-3': ['var(--lfx-text-heading-3-font-size)', 'var(--lfx-text-heading-3-line-height)'] as [
    string,
    string,
  ],
  'heading-4': ['var(--lfx-text-heading-4-font-size)', 'var(--lfx-text-heading-4-line-height)'] as [
    string,
    string,
  ],
  'body-1': ['var(--lfx-text-body-1-font-size)', 'var(--lfx-text-body-1-line-height)'] as [
    string,
    string,
  ],
  'body-2': ['var(--lfx-text-body-2-font-size)', 'var(--lfx-text-body-2-line-height)'] as [
    string,
    string,
  ],
};
