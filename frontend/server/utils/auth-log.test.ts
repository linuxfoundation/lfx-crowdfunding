// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { vi, describe, it, expect, beforeEach, afterEach } from 'vitest';
import { createError } from 'h3';
import { ResponseBodyError } from 'openid-client';
import { isExpectedAuthError, logAuthError } from './auth-log';

const oauthError = (error: string, description?: string) =>
  new ResponseBodyError('server responded with an error in the response body', {
    cause: { error, error_description: description },
    response: new Response(null, { status: 403 }),
  });

describe('auth-log', () => {
  let warn: ReturnType<typeof vi.spyOn>;
  let error: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
    error = vi.spyOn(console, 'error').mockImplementation(() => {});
  });

  afterEach(() => vi.restoreAllMocks());

  it('treats invalid_grant and 4xx H3 errors as expected', () => {
    expect(isExpectedAuthError(oauthError('invalid_grant'))).toBe(true);
    expect(isExpectedAuthError(createError({ statusCode: 400, statusMessage: 'x' }))).toBe(true);
  });

  it('treats other OAuth errors, 5xx and plain errors as unexpected', () => {
    expect(isExpectedAuthError(oauthError('server_error'))).toBe(false);
    expect(isExpectedAuthError(createError({ statusCode: 500, statusMessage: 'x' }))).toBe(false);
    expect(isExpectedAuthError(new Error('network down'))).toBe(false);
  });

  it('logs an expired refresh token as one line with the OAuth code', () => {
    logAuthError(
      'Auth refresh error',
      oauthError('invalid_grant', 'Unknown or invalid refresh token'),
    );
    expect(error).not.toHaveBeenCalled();
    expect(warn).toHaveBeenCalledWith(
      'Auth refresh error (expected) invalid_grant: Unknown or invalid refresh token',
    );
  });

  it('logs a rejected callback as one line, collapsing newlines', () => {
    logAuthError(
      'Auth callback error',
      createError({ statusCode: 400, statusMessage: 'Authentication error: access_denied - a\nb' }),
    );
    expect(error).not.toHaveBeenCalled();
    expect(warn).toHaveBeenCalledWith(
      'Auth callback error (expected) 400: Authentication error: access_denied - a b',
    );
  });

  it('keeps the full error object for unexpected failures', () => {
    const err = new Error('network down');
    logAuthError('Auth refresh error', err);
    expect(warn).not.toHaveBeenCalled();
    expect(error).toHaveBeenCalledWith('Auth refresh error:', err);
  });
});
