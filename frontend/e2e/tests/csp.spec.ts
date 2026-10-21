// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { test, expect, type Page } from '@playwright/test';

// Regression for the CSP nonce XSS (lfx-self-serve-ops#491). The slug used to be reflected
// raw into <meta property="og:url">; since #382 an unknown slug renders the 404 page with
// an encoded og:url, so the raw-attribute case is covered by csp-nonce.server.test.ts.
// This checks the payload still doesn't run and every SSR script carries the header nonce.
// Runs against the production build (csp project): no CSP is sent in dev.
const XSS = '<img src=x onerror=window.__xss=1>';
const PAYLOAD_PATH = `/initiatives/${encodeURIComponent(`x <script y=>${XSS}`)}`;

const nonceOf = (csp: string | undefined) => /'nonce-([^']+)'/.exec(csp ?? '')?.[1];

const blockTranscend = (page: Page) =>
  page.route('**transcend-cdn.com**', (route) => route.abort());

test.describe('CSP nonce', () => {
  test('reflected slug payload does not execute and every script carries the nonce', async ({
    page,
  }) => {
    await blockTranscend(page);
    let dialogs = 0;
    page.on('dialog', (dialog) => {
      dialogs++;
      void dialog.dismiss();
    });

    const response = await page.goto(PAYLOAD_PATH);
    expect(response).not.toBeNull();
    const html = await response!.text();

    await page.waitForLoadState('load');
    expect(
      await page.evaluate(() => (window as unknown as { __xss?: number }).__xss),
    ).toBeUndefined();
    expect(dialogs).toBe(0);

    const nonce = nonceOf(response!.headers()['content-security-policy']);
    expect(nonce).toBeTruthy();

    // Parse the raw HTML, not the live DOM: browsers hide the nonce after load, and
    // strict-dynamic lets loaders add un-nonced scripts later. A DOMParser document has
    // no CSP, so the attribute is readable, and inert `<script` text in attributes
    // isn't counted.
    const scriptNonces = await page.evaluate(
      (source) =>
        [...new DOMParser().parseFromString(source, 'text/html').querySelectorAll('script')].map(
          (script) => script.getAttribute('nonce'),
        ),
      html,
    );
    expect(scriptNonces.length).toBeGreaterThan(0);
    expect(scriptNonces).toEqual(scriptNonces.map(() => nonce));
  });

  // The SSR nonce made unhead miss un-keyed src scripts on hydration and inject them
  // again (airgap.js threw "already initialized"). Each external script must load once.
  test('hydration does not re-inject head scripts', async ({ page }) => {
    await blockTranscend(page);
    await page.goto('/');
    await page.waitForLoadState('load');

    const srcs = await page.evaluate(() =>
      [...document.querySelectorAll('script[src]')].map((script) => script.getAttribute('src')),
    );
    expect(srcs.length).toBeGreaterThan(0);
    expect(srcs).toEqual([...new Set(srcs)]);
  });

  test('each response gets a fresh nonce', async ({ request }) => {
    const nonces = await Promise.all(
      [1, 2].map(async () => {
        const response = await request.get('/initiatives/e2e-test-initiative');
        return nonceOf(response.headers()['content-security-policy']);
      }),
    );
    expect(nonces[0]).toBeTruthy();
    expect(nonces[0]).not.toBe(nonces[1]);
  });
});
