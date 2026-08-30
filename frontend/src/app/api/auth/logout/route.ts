import { cookies } from "next/headers";
import { NextResponse } from "next/server";
import { backendUrl, REFRESH_COOKIE_NAME } from "@/lib/auth/cookie";

/**
 * Called by `AuthProvider.logout()`. Revokes the refresh token server-side
 * and always clears the cookie locally, even if the backend call fails —
 * a stuck cookie the user can never clear from a broken backend is worse
 * than an unrevoked token that will simply expire.
 *
 * No mock-mode branch needed here (unlike `refresh/route.ts`) — the
 * `try/catch` below already degrades correctly when there's no real
 * backend to call: the fetch fails, is swallowed, and the cookie still
 * gets cleared.
 */
export async function POST(request: Request): Promise<Response> {
  const cookieStore = await cookies();
  const refreshToken = cookieStore.get(REFRESH_COOKIE_NAME)?.value;
  const accessAuthHeader = request.headers.get("authorization") ?? undefined;

  if (refreshToken) {
    try {
      await fetch(backendUrl("/api/v1/auth/logout"), {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          ...(accessAuthHeader ? { Authorization: accessAuthHeader } : {}),
        },
        body: JSON.stringify({ refresh_token: refreshToken }),
        cache: "no-store",
      });
    } catch {
      // Best-effort — still clear the local cookie below.
    }
  }

  cookieStore.delete(REFRESH_COOKIE_NAME);
  return new NextResponse(null, { status: 204 });
}
