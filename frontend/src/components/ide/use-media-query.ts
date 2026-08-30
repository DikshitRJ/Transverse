"use client";

import { useSyncExternalStore } from "react";

function subscribe(query: string, callback: () => void): () => void {
  const mql = window.matchMedia(query);
  mql.addEventListener("change", callback);
  return () => mql.removeEventListener("change", callback);
}

/**
 * SSR-safe media query hook (`useSyncExternalStore` — the server snapshot
 * and the first client render both use `serverSnapshot`, so there's no
 * hydration mismatch; it re-syncs to the real value immediately after).
 * Used to switch the split view into stacked tabs below 1024px per the
 * FORGE brief ("below 1024 stack it into tabs rather than letting it
 * break") — defaults to `true` (desktop) since the split view is
 * desktop-first.
 */
export function useMediaQuery(query: string, serverSnapshot = true): boolean {
  return useSyncExternalStore(
    (callback) => subscribe(query, callback),
    () => window.matchMedia(query).matches,
    () => serverSnapshot,
  );
}
