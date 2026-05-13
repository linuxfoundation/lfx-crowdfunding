// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

export enum FundingCategoryId {
  Security = 'security',
  Infrastructure = 'infrastructure',
  Community = 'community',
  Events = 'events',
  Travel = 'travel',
}

export const CATEGORY_HEX: Record<FundingCategoryId, string> = {
  [FundingCategoryId.Security]: '#fe9a00',
  [FundingCategoryId.Infrastructure]: '#0f172b',
  [FundingCategoryId.Community]: '#009aff',
  [FundingCategoryId.Events]: '#00bc7d',
  [FundingCategoryId.Travel]: '#8e51ff',
};
