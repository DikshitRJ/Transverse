/**
 * The one fetch wrapper for the entire app. No component or hook ever calls
 * `fetch()` directly against `/api/v1/*` — everything goes through
 * `apiFetch` (or the typed functions in `endpoints.ts`, which all call it).
 *
 * Responsibilities:
 *  - Prefixes every call with `/api/v1` (same-origin; `next.config.ts`
 *    rewrites this to `BACKEND_URL` in live mode, and MSW intercepts it
 *    transparently at the network layer in mock mode — this file never
 *    branches on `NEXT_PUBLIC_API_MODE`).
 *  - Attaches `Authorization: Bearer <token>` from the in-memory token store.
 *  - Retries idempotent GETs on network failure (not on HTTP error status).
 *  - On a 401, attempts one single-flight silent refresh via
 *    `POST /api/auth/refresh` (our own Next.js Route Handler, which holds
 *    the httpOnly refresh cookie) and retries the original request once.
 *  - Normalises the backend's real error envelope — `{"error": "message"}`,
 *    see `handlers/helpers.go:writeError` — into a typed `ApiError`.
 */

import {
  emitAuthExpired,
  getAccessToken,
  setAccessToken,
} from "@/lib/auth/token-store";
import type { ApiErrorEnvelope } from "@/lib/api/types";

const API_PREFIX = "/api/v1";

export class ApiError extends Error {
  readonly status: number;
  readonly body: unknown;

  constructor(message: string, status: number, body?: unknown) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.body = body;
  }
}

export interface ApiFetchOptions extends Omit<RequestInit, "body"> {
  /** JSON-serializable body. Do not pass a pre-stringified string. */
  json?: unknown;
  /** Raw body escape hatch (e.g. FormData for uploads) — bypasses JSON encoding. */
  body?: BodyInit;
  /** Query string params to append (undefined values are skipped). */
  query?: Record<string, string | number | boolean | undefined>;
  /** Internal — set to true on the retry-after-refresh attempt to prevent loops. */
  _isRetry?: boolean;
  /** Skip the automatic 401-refresh dance (used by auth endpoints themselves). */
  skipAuthRefresh?: boolean;
  /**
   * Skip the `/api/v1` prefix. Only `GET /health` lives outside `/api/v1`
   * on the backend (it's mounted directly on the chi root router in
   * `cmd/server/main.go`, not inside `r.Route("/api/v1", ...)`) — see
   * `getHealth()` in `endpoints.ts`, the sole caller of this flag.
   */
  skipPrefix?: boolean;
}

let refreshInFlight: Promise<string | null> | null = null;

async function refreshAccessToken(): Promise<string | null> {
  if (!refreshInFlight) {
    refreshInFlight = doRefresh().finally(() => {
      refreshInFlight = null;
    });
  }
  return refreshInFlight;
}

async function doRefresh(): Promise<string | null> {
  try {
    const res = await fetch("/api/auth/refresh", {
      method: "POST",
      credentials: "include",
    });
    if (!res.ok) {
      setAccessToken(null);
      emitAuthExpired();
      return null;
    }
    const body = (await res.json()) as { access_token: string };
    setAccessToken(body.access_token);
    return body.access_token;
  } catch {
    setAccessToken(null);
    emitAuthExpired();
    return null;
  }
}

function buildUrl(path: string, query?: ApiFetchOptions["query"], skipPrefix?: boolean): string {
  const prefix = skipPrefix ? "" : API_PREFIX;
  const url = path.startsWith("/") ? `${prefix}${path}` : `${prefix}/${path}`;
  if (!query) return url;
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(query)) {
    if (value !== undefined) params.set(key, String(value));
  }
  const qs = params.toString();
  return qs ? `${url}?${qs}` : url;
}

async function parseErrorBody(res: Response): Promise<{ message: string; body: unknown }> {
  const text = await res.text().catch(() => "");
  if (!text) return { message: res.statusText || `Request failed (${res.status})`, body: undefined };
  try {
    const parsed = JSON.parse(text) as Partial<ApiErrorEnvelope> & Record<string, unknown>;
    if (typeof parsed.error === "string") {
      return { message: parsed.error, body: parsed };
    }
    return { message: text, body: parsed };
  } catch {
    return { message: text, body: undefined };
  }
}

const RETRYABLE_DELAYS_MS = [300, 900];

/**
 * Core request function. Prefer the named functions in `endpoints.ts` —
 * call this directly only for one-off/experimental calls.
 */
export async function apiFetch<T>(path: string, options: ApiFetchOptions = {}): Promise<T> {
  const { json, body, query, headers, _isRetry, skipAuthRefresh, skipPrefix, ...rest } = options;
  const method = (options.method ?? "GET").toUpperCase();
  const url = buildUrl(path, query, skipPrefix);

  const finalHeaders = new Headers(headers);
  const token = getAccessToken();
  if (token && !finalHeaders.has("Authorization")) {
    finalHeaders.set("Authorization", `Bearer ${token}`);
  }

  let finalBody: BodyInit | undefined = body;
  if (json !== undefined) {
    finalHeaders.set("Content-Type", "application/json");
    finalBody = JSON.stringify(json);
  }

  const isIdempotentGet = method === "GET";
  const attempts = isIdempotentGet ? RETRYABLE_DELAYS_MS.length + 1 : 1;

  let lastNetworkError: unknown;
  for (let attempt = 0; attempt < attempts; attempt++) {
    let res: Response;
    try {
      res = await fetch(url, { ...rest, method, headers: finalHeaders, body: finalBody });
    } catch (err) {
      lastNetworkError = err;
      if (attempt < attempts - 1) {
        await new Promise((r) => setTimeout(r, RETRYABLE_DELAYS_MS[attempt]));
        continue;
      }
      throw new ApiError(
        err instanceof Error ? err.message : "Network request failed",
        0,
        undefined,
      );
    }

    if (res.status === 401 && !skipAuthRefresh && !_isRetry) {
      const newToken = await refreshAccessToken();
      if (newToken) {
        return apiFetch<T>(path, { ...options, _isRetry: true });
      }
      const { message, body: errBody } = await parseErrorBody(res);
      throw new ApiError(message, 401, errBody);
    }

    if (!res.ok) {
      const { message, body: errBody } = await parseErrorBody(res);
      throw new ApiError(message, res.status, errBody);
    }

    if (res.status === 204 || res.status === 202) {
      // 202 (async job accepted) bodies are still meaningful (e.g. {job_id}),
      // so only skip parsing on a genuinely empty 204.
      if (res.status === 204) return undefined as T;
    }

    const text = await res.text();
    if (!text) return undefined as T;
    return JSON.parse(text) as T;
  }

  // Unreachable — the loop above always returns or throws — but keeps TS happy.
  throw new ApiError(
    lastNetworkError instanceof Error ? lastNetworkError.message : "Network request failed",
    0,
  );
}

export function apiGet<T>(path: string, options?: ApiFetchOptions): Promise<T> {
  return apiFetch<T>(path, { ...options, method: "GET" });
}

export function apiPost<T>(path: string, json?: unknown, options?: ApiFetchOptions): Promise<T> {
  return apiFetch<T>(path, { ...options, method: "POST", json });
}

export function apiDelete<T>(path: string, options?: ApiFetchOptions): Promise<T> {
  return apiFetch<T>(path, { ...options, method: "DELETE" });
}
