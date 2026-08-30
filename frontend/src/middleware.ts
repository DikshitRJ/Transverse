import { NextResponse, type NextRequest } from "next/server";
import { REFRESH_COOKIE_NAME } from "@/lib/auth/cookie";

/**
 * Route guards (plan.md §2, §5.3 — THRESHOLD owns this file exclusively).
 *
 * The access token lives only in client-side JS memory
 * (`lib/auth/token-store.ts`) — middleware runs on the server/edge and can
 * never see it. The one signal middleware *can* see is the httpOnly
 * `tv_refresh_token` cookie set by `POST /api/auth/session` right after
 * `/auth/callback` completes. Its presence is treated as "this browser has
 * a session worth attempting" — it is not proof the session is still
 * valid (the cookie can outlive a revoked/expired refresh token). That
 * finer-grained check already happens client-side: `AuthProvider` calls
 * `POST /api/auth/refresh` on mount, and `client.ts` does a single-flight
 * silent refresh on any 401; both paths call `emitAuthExpired()` on
 * failure, which clears `user` back to `null`. A page that then still
 * needs to react to "session died mid-visit" should watch
 * `useAuth().isAuthenticated` itself (THRESHOLD's own `/onboarding` layout
 * does this — see `src/app/onboarding/layout.tsx`); this middleware only
 * stops the *first* unauthenticated request to a protected route from
 * ever rendering.
 *
 * `/`, `/signin`, and `/auth/callback` are intentionally never matched
 * below — they must stay public.
 */

export function middleware(request: NextRequest): NextResponse {
  const hasSession = Boolean(request.cookies.get(REFRESH_COOKIE_NAME)?.value);

  if (!hasSession) {
    const signInUrl = new URL("/signin", request.url);
    const next = `${request.nextUrl.pathname}${request.nextUrl.search}`;
    if (next !== "/") signInUrl.searchParams.set("next", next);
    return NextResponse.redirect(signInUrl);
  }

  return NextResponse.next();
}

// Next's build-time config analyzer requires this array to be a literal —
// no `.map()`/computed expressions (it failed the build with "Unsupported
// node type CallExpression" when this was derived from a shared prefix
// list). Keep in sync with the protected-route list in the doc comment
// above by hand.
export const config = {
  matcher: [
    "/dashboard/:path*",
    "/roadmap/:path*",
    "/solve/:path*",
    "/practice/:path*",
    "/profile/:path*",
    "/settings/:path*",
    "/onboarding/:path*",
    // Added during Wave-2 merge: both were missing from the guard list handed
    // to THRESHOLD, but plan.md §2 lists them under "Core application" and both
    // render user-scoped data (`/problems` shows per-user solve state; the
    // tutorial reader marks node completion against the signed-in user). They
    // were reachable unauthenticated — confirmed by a live 200 in the merged
    // smoke test while every other app route correctly returned 307.
    "/problems/:path*",
    "/tutorial/:path*",
  ],
};
