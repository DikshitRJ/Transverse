"use client";

import { useEffect, useState, type ReactNode } from "react";

const IS_MOCK_MODE = process.env.NEXT_PUBLIC_API_MODE === "mock";

/**
 * Starts the browser MSW worker (mocks/browser.ts) before rendering
 * children, so no client component ever fires a real `fetch` against a
 * backend that mock mode isn't running. A no-op (renders immediately) when
 * `NEXT_PUBLIC_API_MODE !== "mock"`. Server-side interception for
 * RSC/Route-Handler fetches is handled separately by `src/instrumentation.ts`.
 */
export function MockProvider({ children }: { children: ReactNode }) {
  const [ready, setReady] = useState(!IS_MOCK_MODE);

  useEffect(() => {
    if (!IS_MOCK_MODE) return;
    let cancelled = false;
    import("@/mocks/browser").then(({ worker }) =>
      worker.start({ onUnhandledRequest: "bypass", quiet: true }).then(() => {
        if (!cancelled) setReady(true);
      }),
    );
    return () => {
      cancelled = true;
    };
  }, []);

  if (!ready) return null;
  return <>{children}</>;
}
