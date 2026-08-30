import { cookies } from "next/headers";
import { NextResponse } from "next/server";
import { REFRESH_COOKIE_MAX_AGE_SECONDS, REFRESH_COOKIE_NAME } from "@/lib/auth/cookie";

/**
 * Called once by `/auth/callback` (the page THRESHOLD builds) right after
 * the OAuth redirect chain lands back on the frontend with tokens in the
 * URL. This route's only job is to take the refresh token out of the URL
 * (where it's visible to browser history/referrers/devtools) and move it
 * into an httpOnly cookie no client-side JS — including a compromised
 * third-party script rendering scraped problem HTML — can read.
 *
 * The access token is NOT handled here: the caller keeps it in memory via
 * `setAccessToken()` directly, it never touches a cookie.
 */
export async function POST(request: Request): Promise<Response> {
  let body: { refresh_token?: string };
  try {
    body = await request.json();
  } catch {
    return NextResponse.json({ error: "invalid request body" }, { status: 400 });
  }

  if (!body.refresh_token) {
    return NextResponse.json({ error: "refresh_token is required" }, { status: 400 });
  }

  const cookieStore = await cookies();
  cookieStore.set(REFRESH_COOKIE_NAME, body.refresh_token, {
    httpOnly: true,
    secure: process.env.NODE_ENV === "production",
    sameSite: "lax",
    path: "/",
    maxAge: REFRESH_COOKIE_MAX_AGE_SECONDS,
  });

  return new NextResponse(null, { status: 204 });
}
