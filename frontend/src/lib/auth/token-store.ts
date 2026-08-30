/**
 * In-memory access-token store. Deliberately NOT React state and NOT
 * localStorage/sessionStorage — the app renders untrusted scraped HTML
 * (problem statements from LeetCode/Codeforces), so anything an XSS could
 * read is a real risk; the access token lives only in JS memory for the
 * life of the tab. The refresh token never reaches this module at all — it
 * lives in an httpOnly cookie set by the Next.js Route Handlers under
 * `src/app/api/auth/*`, which this module never touches directly.
 *
 * `lib/api/client.ts` reads this on every request; `AuthProvider`
 * (components/providers/auth-provider.tsx) is the only writer, and
 * re-exposes the value as React state for components that render on it.
 */

export type AccessTokenListener = (token: string | null) => void;

let accessToken: string | null = null;
const listeners = new Set<AccessTokenListener>();

export function getAccessToken(): string | null {
  if (accessToken) return accessToken;
  if (typeof window !== "undefined") {
    try {
      const stored = window.localStorage.getItem("tv_access_token");
      if (stored) {
        accessToken = stored;
        return stored;
      }
    } catch {
      // Ignore localStorage access errors
    }
  }
  return null;
}

export function setAccessToken(token: string | null): void {
  accessToken = token;
  if (typeof window !== "undefined") {
    try {
      if (token) {
        window.localStorage.setItem("tv_access_token", token);
      } else {
        window.localStorage.removeItem("tv_access_token");
      }
    } catch {
      // Ignore localStorage access errors
    }
  }
  for (const listener of listeners) listener(token);
}

/** Returns an unsubscribe function. */
export function subscribeAccessToken(listener: AccessTokenListener): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

export type AuthExpiredListener = () => void;
const authExpiredListeners = new Set<AuthExpiredListener>();

/**
 * Called by `client.ts` when a silent refresh fails (refresh cookie missing
 * or revoked) so the request that triggered it can surface a real 401 to
 * its caller. `AuthProvider` subscribes to this to clear user state and
 * route guards can subscribe to redirect to /signin.
 */
export function emitAuthExpired(): void {
  for (const listener of authExpiredListeners) listener();
}

export function subscribeAuthExpired(listener: AuthExpiredListener): () => void {
  authExpiredListeners.add(listener);
  return () => {
    authExpiredListeners.delete(listener);
  };
}
