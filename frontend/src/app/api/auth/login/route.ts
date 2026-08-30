import { cookies } from "next/headers";
import { NextResponse } from "next/server";
import {
  backendUrl,
  isMockMode,
  mockToken,
  REFRESH_COOKIE_MAX_AGE_SECONDS,
  REFRESH_COOKIE_NAME,
} from "@/lib/auth/cookie";
import type { AuthResponse, LoginRequest } from "@/lib/api/types";
import { rawUser } from "@/mocks/fixtures/user";

export async function POST(request: Request): Promise<Response> {
  let body: LoginRequest;
  try {
    body = await request.json();
  } catch {
    return NextResponse.json({ error: "Invalid JSON payload" }, { status: 400 });
  }

  if (!body.email || !body.password) {
    return NextResponse.json({ error: "Email and password are required" }, { status: 400 });
  }

  const cookieStore = await cookies();

  if (isMockMode()) {
    cookieStore.set(REFRESH_COOKIE_NAME, mockToken("mock-refresh"), {
      httpOnly: true,
      secure: process.env.NODE_ENV === "production",
      sameSite: "lax",
      path: "/",
      maxAge: REFRESH_COOKIE_MAX_AGE_SECONDS,
    });
    return NextResponse.json({
      access_token: mockToken("mock-access"),
      expires_in: 3600,
      user: { ...rawUser, email: body.email },
    });
  }

  let res: Response;
  try {
    res = await fetch(backendUrl("/api/v1/auth/login"), {
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
      { error: errorJson.error || "Login failed" },
      { status: res.status },
    );
  }

  const data = (await res.json()) as AuthResponse;

  if (data.refresh_token) {
    cookieStore.set(REFRESH_COOKIE_NAME, data.refresh_token, {
      httpOnly: true,
      secure: process.env.NODE_ENV === "production",
      sameSite: "lax",
      path: "/",
      maxAge: REFRESH_COOKIE_MAX_AGE_SECONDS,
    });
  }

  return NextResponse.json({
    access_token: data.access_token,
    expires_in: data.expires_in,
    user: data.user,
  });
}
