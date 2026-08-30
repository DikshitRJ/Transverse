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
import type { AuthResponse, RegisterRequest } from "@/lib/api/types";
import { rawUser } from "@/mocks/fixtures/user";

export async function POST(request: Request): Promise<Response> {
  let body: RegisterRequest;
  try {
    body = await request.json();
  } catch {
    return NextResponse.json({ error: "Invalid JSON payload" }, { status: 400 });
  }

  if (!body.email || !body.password) {
    return NextResponse.json({ error: "Email and password are required" }, { status: 400 });
  }

  const cookieStore = await cookies();
  const secure = isCookieSecure(request);

  if (isMockMode()) {
    cookieStore.set(REFRESH_COOKIE_NAME, mockToken("mock-refresh"), {
      httpOnly: true,
      secure,
      sameSite: "lax",
      path: "/",
      maxAge: REFRESH_COOKIE_MAX_AGE_SECONDS,
    });
    const resp = NextResponse.json({
      access_token: mockToken("mock-access"),
      expires_in: 3600,
      user: {
        ...rawUser,
        email: body.email,
        username: body.username || body.email.split("@")[0],
      },
    });
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
    res = await fetch(backendUrl("/api/v1/auth/register"), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
      cache: "no-store",
    });
  } catch {
    return NextResponse.json({ error: "Backend service unreachable" }, { status: 502 });
  }

  if (!res.ok) {
    const errorJson = await res.json().catch(async () => ({ error: await res.text() }));
    return NextResponse.json(
      { error: errorJson.error || "Registration failed" },
      { status: res.status },
    );
  }

  const data = (await res.json()) as AuthResponse;

  if (data.refresh_token) {
    cookieStore.set(REFRESH_COOKIE_NAME, data.refresh_token, {
      httpOnly: true,
      secure,
      sameSite: "lax",
      path: "/",
      maxAge: REFRESH_COOKIE_MAX_AGE_SECONDS,
    });
  }

  const response = NextResponse.json({
    access_token: data.access_token,
    expires_in: data.expires_in,
    user: data.user,
  });

  if (data.refresh_token) {
    response.cookies.set(REFRESH_COOKIE_NAME, data.refresh_token, {
      httpOnly: true,
      secure,
      sameSite: "lax",
      path: "/",
      maxAge: REFRESH_COOKIE_MAX_AGE_SECONDS,
    });
  }

  return response;
}
