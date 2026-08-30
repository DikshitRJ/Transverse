/**
 * A same-tab handoff cache for `CloseSessionResponse`.
 *
 * `POST /practice/close` (`closePracticeSession`) is the only place
 * `per_topic_breakdown` / `mastery_score` / `accuracy` ever get computed —
 * `GET /practice/session/{id}` (`GetSessionResponse`) never returns them
 * (see `CloseSessionResponse` vs `GetSessionResponse` in lib/api/types.ts).
 * Since Next.js's router has no built-in way to hand a payload to the next
 * route, and re-calling close a second time isn't a safe assumption against
 * a real backend, whichever screen actually calls close (the quiz page, or
 * "End session" on `/practice`) stashes the response here, keyed by session
 * id, immediately before navigating to the page that renders it
 * (`/onboarding/results`, `/practice/session/[id]`). Those pages read the
 * cache once on mount and degrade gracefully if it's missing (e.g. a user
 * opens the summary link directly, or reloads) — see each page for its
 * fallback.
 *
 * `sessionStorage`, not a global store: this is a one-shot, same-tab
 * handoff for data the user just generated, not shared or persisted state.
 */
import type { CloseSessionResponse } from "@/lib/api/types";

const PREFIX = "tv-practice-close:";

export function cacheCloseResult(sessionId: string, result: CloseSessionResponse): void {
  if (typeof window === "undefined") return;
  try {
    window.sessionStorage.setItem(`${PREFIX}${sessionId}`, JSON.stringify(result));
  } catch {
    // sessionStorage can throw (private mode, quota) — the read side just falls back.
  }
}

export function readCachedCloseResult(sessionId: string): CloseSessionResponse | null {
  if (typeof window === "undefined") return null;
  try {
    const raw = window.sessionStorage.getItem(`${PREFIX}${sessionId}`);
    if (!raw) return null;
    return JSON.parse(raw) as CloseSessionResponse;
  } catch {
    return null;
  }
}
