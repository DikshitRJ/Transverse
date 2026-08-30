import type { NextConfig } from "next";
import path from "node:path";

const BACKEND_URL = process.env.BACKEND_URL ?? "http://localhost:8080";

const nextConfig: NextConfig = {
  output: "standalone",
  // The repo root (C:\Users\aksha\Transverse) has its own package.json/lockfile
  // (root-level tooling, unrelated to this app) which Next's workspace-root
  // inference picks up as an ambiguous second lockfile. Pin the trace root to
  // this app explicitly — matters for `output: "standalone"`, which uses file
  // tracing to decide what ships in the Docker image (see HARNESS's Dockerfile).
  outputFileTracingRoot: path.join(__dirname),

  // `msw`/`@mswjs/interceptors` use Node's `exports` map "node" condition
  // (see @mswjs/interceptors's package.json) to pick their server-runtime
  // entry points. Webpack's default resolver doesn't set that condition,
  // so bundling them (the default) throws
  // "Package path ./ClientRequest is not exported" the moment
  // `src/instrumentation.ts` imports `mocks/server.ts` in mock mode.
  // Marking them external makes Next use real Node `require`/`import`
  // resolution for these two packages instead, which resolves correctly.
  serverExternalPackages: ["msw", "@mswjs/interceptors"],

  webpack: (config, { isServer }) => {
    if (isServer) {
      const externals = Array.isArray(config.externals) ? config.externals : [];
      externals.push("msw", "@mswjs/interceptors");
      config.externals = externals;
    }
    return config;
  },

  async rewrites() {
    return [
      // Every backend route the app calls goes through this same-origin
      // proxy — no CORS preflight, no backend change needed. In mock mode
      // (NEXT_PUBLIC_API_MODE=mock) MSW intercepts these requests before
      // they ever leave the browser, so this rewrite is simply unused, not
      // a conflict.
      {
        source: "/api/v1/:path*",
        destination: `${BACKEND_URL}/api/v1/:path*`,
      },
      // GET /health is mounted directly on the backend's root chi router
      // (cmd/server/main.go), NOT under /api/v1 — it needs its own rule.
      {
        source: "/health",
        destination: `${BACKEND_URL}/health`,
      },
    ];
  },
};

export default nextConfig;

/**
 * SSE verification (plan.md §5.1) — a buffered `/events/stream` is a
 * silent failure: the browser just never receives events until the
 * connection eventually closes, with no error surfaced anywhere.
 *
 * `next.config.ts` rewrites are a proxy pass-through, not a route handler,
 * so Next does not buffer the response body itself — but the OS/reverse
 * proxy in front of it (e.g. nginx in the compose stack) can. Check with:
 *
 *   curl -N http://localhost:3000/api/v1/events/stream \
 *     -H "Authorization: Bearer <token>"
 *
 * `-N` disables curl's own output buffering. You should see the initial
 * `event: connected` frame land immediately, then a `: keepalive` comment
 * every 15s (from realtime/handler.go's heartbeat ticker) even with no
 * real events firing. If output only appears in one burst when the
 * connection finally times out or closes, something upstream is buffering
 * — check `X-Accel-Buffering: no` is set (or the equivalent for whatever
 * sits in front of Next in production) and that gzip compression isn't
 * applied to this route.
 */
