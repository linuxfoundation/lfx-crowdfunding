// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { useRuntimeConfig } from 'nuxt/app';

export interface IntercomBootOptions {
  app_id?: string;
  user_id?: string;
  name?: string;
  email?: string;
  intercom_user_jwt?: string;
  [key: string]: string | number | boolean | undefined;
}

declare global {
  interface Window {
    Intercom?: (...args: unknown[]) => void;
    intercomSettings?: Record<string, unknown>;
  }
}

// Module-level singletons — one Intercom instance per page, shared across all composable calls.
let isLoaded = false;
let isBooted = false;
let isLoading = false;
let bootedWithIdentity = false;

function loadIntercomScript(appId: string): void {
  if (isLoaded || isLoading || typeof window === 'undefined') return;
  isLoading = true;

  // Create stub so queued commands work before script loads
  const w = window as Record<string, unknown>;
  const ic = w.Intercom;
  if (typeof ic === 'function') {
    (ic as (...args: unknown[]) => void)('reattach_activator');
    (ic as (...args: unknown[]) => void)('update', window.intercomSettings);
  } else {
    const stub = (...args: unknown[]) => {
      (stub as unknown as { q: unknown[][] }).q.push(args);
    };
    (stub as unknown as { q: unknown[][] }).q = [];
    w.Intercom = stub as (...args: unknown[]) => void;
  }

  window.intercomSettings = {
    api_base: 'https://api-iam.intercom.io',
    app_id: appId,
  };

  const script = document.createElement('script');
  script.type = 'text/javascript';
  script.async = true;
  script.src = `https://widget.intercom.io/widget/${appId}`;
  script.onload = () => {
    isLoaded = true;
    isLoading = false;
  };
  script.onerror = () => {
    isLoading = false;
    console.error('[useIntercom] Failed to load Intercom script');
  };

  const first = document.getElementsByTagName('script')[0];
  if (first?.parentNode) {
    first.parentNode.insertBefore(script, first);
  } else {
    (document.head || document.body).appendChild(script);
  }
}

function shutdownForReboot(): void {
  if (typeof window !== 'undefined' && window.Intercom) {
    try {
      window.Intercom('shutdown');
    } catch {
      // ignore
    }
  }
  isBooted = false;
  bootedWithIdentity = false;
}

export const useIntercom = () => {
  const {
    public: { intercomAppId },
  } = useRuntimeConfig();

  const appId = intercomAppId as string | undefined;

  function boot(options: IntercomBootOptions): Promise<void> {
    return new Promise((resolve, reject) => {
      if (typeof window === 'undefined') {
        reject(new Error('[useIntercom] Window is undefined'));
        return;
      }
      if (!appId) {
        reject(new Error('[useIntercom] No Intercom app ID configured'));
        return;
      }

      if (isBooted) {
        if (options.user_id && !bootedWithIdentity) {
          // Upgrade anonymous → identified: shutdown and re-boot with identity
          shutdownForReboot();
        } else {
          // Same mode — just update
          const { intercom_user_jwt: _jwt, app_id: _aid, ...rest } = options;
          update(rest);
          resolve();
          return;
        }
      }

      if (!isLoaded && !isLoading) {
        loadIntercomScript(appId);
      }

      // JWT must be set in intercomSettings before boot()
      if (options.intercom_user_jwt) {
        window.intercomSettings = window.intercomSettings ?? {};
        window.intercomSettings.intercom_user_jwt = options.intercom_user_jwt;
      }

      const checkLoaded = setInterval(() => {
        if (!isLoaded || !window.Intercom) return;

        clearInterval(checkLoaded);
        clearTimeout(timeoutHandle);

        if (isBooted) {
          if (options.user_id && !bootedWithIdentity) {
            shutdownForReboot();
            // Fall through to boot below
          } else {
            const { intercom_user_jwt: _jwt, app_id: _aid, ...rest } = options;
            update(rest);
            resolve();
            return;
          }
        }

        isBooted = true;
        try {
          const { intercom_user_jwt: _jwt, ...bootOpts } = options;
          window.Intercom!('boot', {
            api_base: 'https://api-iam.intercom.io',
            app_id: appId,
            ...bootOpts,
          });
          bootedWithIdentity = !!bootOpts.user_id;

          if (bootOpts.user_id) {
            try {
              window.Intercom!('update', {
                user_id: bootOpts.user_id,
                name: bootOpts.name,
                email: bootOpts.email,
              });
            } catch {
              // boot succeeded — update failure is non-fatal
            }
          }
          resolve();
        } catch (err) {
          isBooted = false;
          console.error('[useIntercom] Boot failed', err);
          reject(err);
        }
      }, 100);

      const timeoutHandle = setTimeout(() => {
        clearInterval(checkLoaded);
        if (!isBooted) {
          isLoading = false;
          reject(new Error('[useIntercom] Script failed to load — check network or CSP'));
        }
      }, 10000);
    });
  }

  function update(data?: Partial<IntercomBootOptions>): void {
    if (typeof window !== 'undefined' && window.Intercom && isBooted) {
      try {
        window.Intercom('update', data ?? {});
      } catch (err) {
        console.error('[useIntercom] Update failed', err);
      }
    }
  }

  function shutdown(): void {
    if (typeof window === 'undefined') return;
    if (window.intercomSettings?.intercom_user_jwt) {
      delete window.intercomSettings.intercom_user_jwt;
    }
    if (window.Intercom && isBooted) {
      try {
        window.Intercom('shutdown');
      } catch (err) {
        console.error('[useIntercom] Shutdown failed', err);
      }
    }
    isBooted = false;
    bootedWithIdentity = false;
  }

  return { boot, update, shutdown };
};
