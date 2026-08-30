/**
 * Best-effort `problem_id -> topic` lookup for client-side aggregation.
 *
 * `GetSessionResponse.responses` (GET /practice/session/{id}) only carries
 * `problem_id`, not the topic — there is no endpoint that returns a
 * session's per-response topic breakdown after the fact (only
 * `CloseSessionResponse.per_topic_breakdown`, computed once at close time
 * and not persisted for later re-fetch). The session summary page
 * (`/practice/session/[id]`) needs topic buckets to chart even when a
 * viewer lands on it without having just closed the session in this tab
 * (see that page for the sessionStorage-cache-first strategy).
 *
 * In mock mode this resolves accurately via the same fixture the mock
 * handlers use. Against a live backend this fixture won't contain real
 * problem ids, so lookups fall back to "unknown" — an honest degradation
 * given no endpoint provides this mapping, documented in PULSE's final
 * report rather than silently guessed at.
 */
import { PROBLEMS } from "@/mocks/fixtures/problems";

const PROBLEM_TOPIC_BY_ID = new Map(PROBLEMS.map((p) => [p.id, p.topic] as const));

export function lookupTopicForProblem(problemId: string): string {
  return PROBLEM_TOPIC_BY_ID.get(problemId) ?? "unknown";
}
