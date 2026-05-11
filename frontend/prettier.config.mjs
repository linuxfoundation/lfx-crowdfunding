// Copyright (c) 2025 The Linux Foundation and each contributor.
// SPDX-License-Identifier: MIT

/** @type {import("prettier").Config} */
export default {
  semi: true,
  trailingComma: 'all',
  singleQuote: true,
  printWidth: 100,
  tabWidth: 2,
  useTabs: false,
  vueIndentScriptAndStyle: false,
  htmlWhitespaceSensitivity: 'css',
  singleAttributePerLine: true,
  bracketSpacing: true,
  bracketSameLine: false,
  arrowParens: 'always',
  endOfLine: 'lf',
  overrides: [
    {
      files: ['*.vue'],
      options: {
        semi: true,
        singleAttributePerLine: true,
        htmlWhitespaceSensitivity: 'css',
        printWidth: 120,
      },
    },
    {
      files: ['*.json', '*.jsonc'],
      options: {
        trailingComma: 'none',
      },
    },
    {
      files: ['*.md', '*.mdx'],
      options: {
        proseWrap: 'preserve',
        htmlWhitespaceSensitivity: 'ignore',
      },
    },
    {
      files: ['*.yaml', '*.yml'],
      options: {
        tabWidth: 2,
        singleQuote: false,
      },
    },
  ],
};
