// Copyright (c) 2025 The Linux Foundation and each contributor.
// SPDX-License-Identifier: MIT
import type { Config } from 'tailwindcss';

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
    extend: {
      colors: {
        brand: {
          50: '#eef5ff',
          100: '#d9e8ff',
          200: '#bcd6ff',
          300: '#8ebbff',
          400: '#5995ff',
          500: '#3170ff',
          600: '#1a4ff5',
          700: '#1340e1',
          800: '#1635b6',
          900: '#182f8f',
        },
      },
      fontFamily: {
        primary: ['var(--lfx-font-primary, Inter)', 'sans-serif'],
      },
    },
  },
  plugins: [],
} satisfies Config;
