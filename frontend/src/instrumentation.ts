/**
 * Next.js instrumentation hook — intentionally a no-op.
 *
 * WHY THIS DOES NOTHING (do not "restore" the MSW startup without reading this):
 *
 * This hook originally started the Node-side MSW server in mock mode, so that
 * server-issued requests (RSC data fetches and the `src/app/api/auth/*` Route
 * Handlers calling `BACKEND_URL`) were intercepted too. It could not be made
 * to work, and while it was here the dev server did not boot at all.
 *
 * `msw/node` imports `@mswjs/interceptors/ClientRequest`, which that package
 * only exposes under the "node" export condition. Three approaches were tried
 * and all failed:
 *
 *   1. Plain dynamic `import("./mocks/server")` — webpack bundles it and
 *      throws at compile time:
 *      "Package path ./ClientRequest is not exported from @mswjs/interceptors".
 *      `serverExternalPackages` + a webpack `externals` hook do NOT fix this;
 *      Next compiles `instrumentation.ts` in a separate pass they don't reach.
 *   2. `webpackIgnore`/`turbopackIgnore` pragmas — dodges the compile error by
 *      emitting a raw runtime import, but that then resolves relative to the
 *      COMPILED output (`.next/server/instrumentation.js` -> `.next/server/mocks/server`),
 *      a path the build never emits. Result: `ERR_MODULE_NOT_FOUND` at boot,
 *      killing `next dev` and `next start` outright in mock mode.
 *   3. Adding "node" to webpack's `resolve.conditionNames` — clears the
 *      "not exported" error, but webpack then tries to bundle node builtins
 *      and fails on `node:https`.
 *
 * WHAT WE DO INSTEAD: mock mode is browser-side only, via `MockProvider` ->
 * `src/mocks/browser.ts`. That covers every request the UI actually issues,
 * because all data fetching in this app goes through client-side TanStack
 * Query hooks in `src/lib/api`. `src/mocks/server.ts` still exists and is
 * still used directly by Vitest setup, where plain Node resolution applies
 * and none of the above is a problem.
 *
 * CONSEQUENCE TO KNOW ABOUT: a Server Component that fetches directly from
 * `BACKEND_URL` will NOT be intercepted in mock mode — it will attempt a real
 * network call and fail if no backend is running. If you need server-side
 * data in mock mode, either fetch it client-side through the existing hooks,
 * or branch on `process.env.NEXT_PUBLIC_API_MODE` in the Route Handler and
 * return a fixture directly.
 *
 * Production (`NEXT_PUBLIC_API_MODE=live`) was never affected by any of this:
 * the old hook returned before reaching the import, and the standalone server
 * boots and serves correctly.
 */
export async function register(): Promise<void> {
  // Intentionally empty. See the comment above before adding anything here.
}
