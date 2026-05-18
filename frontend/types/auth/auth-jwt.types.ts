// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

export interface DecodedOidcToken {
  sub: string;
  name?: string;
  email?: string;
  picture?: string;
  email_verified?: boolean;
  updated_at?: string;
  username?: string;
  iss: string;
  iat: number;
  exp: number;
}

export interface DecodedIdToken {
  sub: string;
  name?: string;
  email?: string;
  picture?: string;
  email_verified?: boolean;
  updated_at?: string;
  iss?: string;
  aud?: string | string[];
  iat?: number;
  exp?: number;
  [key: string]: string[] | string | number | boolean | undefined;
}
