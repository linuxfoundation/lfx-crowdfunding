// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

export interface DocSection {
  slug: string;
  title: string;
  description: string;
  displayOrder: number;
  tags: string[];
  lastUpdated: string | null;
  children: DocSection[];
}

export interface DocSectionsResponse {
  sections: DocSection[];
}

export interface DocSearchDocument {
  slug: string;
  title: string;
  description: string;
  content: string;
}

export interface DocSearchResult {
  id: string;
  score: number;
  slug: string;
  title: string;
  description: string;
}

export interface DocArticle {
  slug: string;
  title: string;
  description: string;
  bodyHtml: string;
  tags: string[];
  lastUpdated: string | null;
}
