/** Shared constants for the httpOnly refresh-token cookie set by the Route Handlers in `src/app/api/auth/*`. */

export const REFRESH_COOKIE_NAME = "tv_refresh_token";

/** 30 days — the cookie's own ceiling; the backend's own refresh-token TTL/revocation is still authoritative. */
export const REFRESH_COOKIE_MAX_AGE_SECONDS = 60 * 60 * 24 * 30;

export function backendUrl(path: string): string {
  const base = process.env.BACKEND_URL ?? "http://localhost:8080";
  return `${base}${path}`;
}

/**
 * These 3 Route Handlers (session/refresh/logout) are the only server-side
 * code in this app that talks to the backend directly (not through
 * `lib/api`, and not proxied by `next.config.ts`'s rewrites). In mock mode
 * that direct `fetch(backendUrl(...))` call needs to be short-circuited
 * here rather than intercepted by MSW — Next's `instrumentation.ts`
 * bundles through a compiler pass that does not honor
 * `@mswjs/interceptors`'s "node" package-export condition (reproduced with
 * `serverExternalPackages`, a manual `webpack()` externals hook, AND a
 * `webpackIgnore` escape hatch — none of the standard fixes reach this
 * specific pass in this Next version), so `msw/node`'s `setupServer()`
 * cannot run there. Client-side MSW (`MockProvider` -> `mocks/browser.ts`)
 * is unaffected and still intercepts every other request normally — this
 * only covers the narrow gap these 3 routes would otherwise have.
 */
export function isMockMode(): boolean {
  return process.env.NEXT_PUBLIC_API_MODE === "mock";
}

export function mockToken(prefix: string): string {
  return `${prefix}-${Date.now()}-${Math.random().toString(36).slice(2, 10)}`;
}
