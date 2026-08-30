import { cookies } from "next/headers";
import { NextResponse } from "next/server";
import {
  backendUrl,
  isCookieSecure,
  isMockMode,
  mockToken,
  REFRESH_COOKIE_MAX_AGE_SECONDS,
  REFRESH_COOKIE_NAME,
} from "@/lib/auth/cookie";
import type { AuthTokenResponse } from "@/lib/api/types";

/**
 * The silent-refresh endpoint `client.ts` calls on a 401. Runs server-side
 * so it can read the httpOnly refresh cookie (client JS never can). Calls
 * the backend's `POST /auth/refresh` directly against `BACKEND_URL`
 * (server-to-server — this does NOT go through the `/api/v1` browser
 * rewrite, which only exists for browser-origin requests) with
 * `{refresh_token}` in the body, rotates the cookie to the NEW refresh
 * token the backend issues (the backend revokes the old one on every use —
 * see `AuthHandler.Refresh`), and returns only `{access_token, expires_in}`
 * to the browser. The refresh token itself never reaches client JS.
 *
 * In mock mode this short-circuits entirely (see `isMockMode()` in
 * `lib/auth/cookie.ts`) rather than relying on MSW to intercept the
 * backend call — a Next.js/webpack limitation prevents MSW's node
 * interceptor from loading via `instrumentation.ts` in this project as of
 * this writing.
 */
export async function POST(request: Request): Promise<Response> {
  const cookieStore = await cookies();
  const refreshToken = cookieStore.get(REFRESH_COOKIE_NAME)?.value;
  const secure = isCookieSecure(request);

  if (!refreshToken) {
    return NextResponse.json({ error: "no refresh session" }, { status: 401 });
  }

  // Mock mode: fabricate a token pair instead of calling the backend — see
  // the doc comment on `isMockMode()` for why this can't just be MSW.
  // `mocks/handlers.ts`'s protected routes only check that *some*
  // non-empty Bearer token was sent, not its value, so this is sufficient
  // to keep the whole session working end-to-end against the mock layer.
  if (isMockMode()) {
    cookieStore.set(REFRESH_COOKIE_NAME, mockToken("mock-refresh"), {
      httpOnly: true,
      secure,
      sameSite: "lax",
      path: "/",
      maxAge: REFRESH_COOKIE_MAX_AGE_SECONDS,
    });
    const resp = NextResponse.json({ access_token: mockToken("mock-access"), expires_in: 3600 });
    resp.cookies.set(REFRESH_COOKIE_NAME, mockToken("mock-refresh"), {
      httpOnly: true,
      secure,
      sameSite: "lax",
      path: "/",
      maxAge: REFRESH_COOKIE_MAX_AGE_SECONDS,
    });
    return resp;
  }

  let res: Response;
  try {
    res = await fetch(backendUrl("/api/v1/auth/refresh"), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: refreshToken }),
      cache: "no-store",
    });
  } catch {
    return NextResponse.json({ error: "backend unreachable" }, { status: 502 });
  }

  if (!res.ok) {
    // Refresh token is dead (expired/revoked) — clear the cookie so we don't keep retrying it.
    cookieStore.delete(REFRESH_COOKIE_NAME);
    const message = await res.text().catch(() => "refresh failed");
    const resp = NextResponse.json({ error: message || "refresh failed" }, { status: res.status });
    resp.cookies.delete(REFRESH_COOKIE_NAME);
    return resp;
  }

  const body = (await res.json()) as AuthTokenResponse;

  cookieStore.set(REFRESH_COOKIE_NAME, body.refresh_token, {
    httpOnly: true,
    secure,
    sameSite: "lax",
    path: "/",
    maxAge: REFRESH_COOKIE_MAX_AGE_SECONDS,
  });

  const response = NextResponse.json({
    access_token: body.access_token,
    expires_in: body.expires_in,
  });
  response.cookies.set(REFRESH_COOKIE_NAME, body.refresh_token, {
    httpOnly: true,
    secure,
    sameSite: "lax",
    path: "/",
    maxAge: REFRESH_COOKIE_MAX_AGE_SECONDS,
  });

  return response;
}
