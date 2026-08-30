import { NextResponse } from "next/server";

/**
 * Liveness probe for THIS container — deliberately self-contained.
 *
 * The Docker HEALTHCHECK originally targeted `/health`, but that path is not a
 * route in this app: `next.config.ts` rewrites `/health` to
 * `${BACKEND_URL}/health`, i.e. the Go backend. That made the frontend's health
 * a function of the backend's — the container would be marked unhealthy while
 * serving pages perfectly well, purely because the backend was down or still
 * starting, and `restart: unless-stopped` would then churn it.
 *
 * A container healthcheck must answer "is THIS process able to serve?" and
 * nothing else. Backend reachability is the backend's own healthcheck to report.
 *
 * Kept out of the `/api/v1/*` namespace so it is never captured by the
 * backend-proxy rewrite.
 */
export const dynamic = "force-dynamic";

export function GET() {
  return NextResponse.json({ status: "ok", service: "transverse-frontend" }, { status: 200 });
}
