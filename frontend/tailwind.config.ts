// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT
import type { Config } from 'tailwindcss';
import { lfxColors } from './app/config/styles/colors';
import { lfxFontSizes } from './app/config/styles/font-size';

export default {
  content: ['./app/**/*.{vue,ts,js}', './server/**/*.{ts,js}'],
  theme: {
    screens: {
      sm: '640px',
      md: '768px',
      lg: '1024px',
      xl: '1280px',
      '2xl': '1536px',
    },
    colors: lfxColors,
    fontFamily: {
      primary: ['var(--lfx-font-primary)', 'sans-serif'],
      secondary: ['var(--lfx-font-secondary)', 'sans-serif'],
    },
    fontSize: lfxFontSizes,
    boxShadow: {
      none: 'var(--lfx-shadow-none)',
      xs: 'var(--lfx-shadow-xs)',
      sm: 'var(--lfx-shadow-sm)',
      md: 'var(--lfx-shadow-md)',
      lg: 'var(--lfx-shadow-lg)',
      xl: 'var(--lfx-shadow-xl)',
      '2xl': 'var(--lfx-shadow-2xl)',
    },
    borderRadius: {
      none: '0',
      xs: 'var(--lfx-radius-xs)',
      sm: 'var(--lfx-radius-sm)',
      md: 'var(--lfx-radius-md)',
      lg: 'var(--lfx-radius-lg)',
      xl: 'var(--lfx-radius-xl)',
      '2xl': 'var(--lfx-radius-2xl)',
      '3xl': 'var(--lfx-radius-3xl)',
      full: '9999px',
    },
    extend: {
      outlineWidth: {
        3: '0.1875rem',
      },
      height: {
        7.5: '1.875rem',
        17: '4.25rem',
        18: '4.5rem',
        29: '7.5rem',
      },
      width: {
        13: '3.25rem',
        51: '12.5rem',
        70: '17.5rem',
        78: '19.5rem',
        90: '22.5rem',
        100: '25rem',
      },
      spacing: {
        13: '3.25rem',
        15: '3.75rem',
        17: '4.25rem',
        18: '4.5rem',
        21: '5.25rem',
        22: '5.5rem',
        25: '6.25rem',
        27: '6.75rem',
        30: '7.5rem',
      },
    },
  },
  plugins: [],
} satisfies Config;
