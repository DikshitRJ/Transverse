/**
 * `PracticeSession.scope` and `.responses` are raw, undecoded JSONB on the
 * wire (see `lib/api/types.ts`'s comment on `PracticeSession`) — parsed
 * defensively here rather than trusted as `SessionScope`/`SessionResponse[]`
 * directly, same caveat FOUNDATION.md calls out for `User.dna`.
 */
import type { PracticeSession } from "@/lib/api/types";

export interface ParsedSessionResponse {
  problem_id: string;
  is_correct: boolean;
  skipped: boolean;
  time_taken_ms: number;
  theta_before: number;
  theta_after: number;
  submitted_at: string;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

export function parseSessionResponses(raw: unknown): ParsedSessionResponse[] {
  if (!Array.isArray(raw)) return [];
  const out: ParsedSessionResponse[] = [];
  for (const item of raw) {
    if (!isRecord(item)) continue;
    out.push({
      problem_id: typeof item.problem_id === "string" ? item.problem_id : "",
      is_correct: typeof item.is_correct === "boolean" ? item.is_correct : false,
      skipped: typeof item.skipped === "boolean" ? item.skipped : false,
      time_taken_ms: typeof item.time_taken_ms === "number" ? item.time_taken_ms : 0,
      theta_before: typeof item.theta_before === "number" ? item.theta_before : 0,
      theta_after: typeof item.theta_after === "number" ? item.theta_after : 0,
      submitted_at: typeof item.submitted_at === "string" ? item.submitted_at : "",
    });
  }
  return out;
}

export function parseSessionTopics(scope: unknown): string[] {
  if (!isRecord(scope)) return [];
  const topics = scope.topics;
  if (!Array.isArray(topics)) return [];
  return topics.filter((t): t is string => typeof t === "string");
}

export interface SessionSummary {
  session: PracticeSession;
  responses: ParsedSessionResponse[];
  topics: string[];
  correctCount: number;
  skippedCount: number;
  attempted: number;
  /** 0–1, correct / attempted (skips excluded from the denominator). */
  accuracy: number;
}

export function summarizeSession(session: PracticeSession): SessionSummary {
  const responses = parseSessionResponses(session.responses);
  const topics = parseSessionTopics(session.scope);
  const skippedCount = responses.filter((r) => r.skipped).length;
  const correctCount = responses.filter((r) => !r.skipped && r.is_correct).length;
  const attempted = responses.length - skippedCount;
  const accuracy = attempted > 0 ? correctCount / attempted : 0;
  return { session, responses, topics, correctCount, skippedCount, attempted, accuracy };
}

/** Badge-friendly accuracy tier — mirrors the same thresholds used for difficulty severity elsewhere in PRISM. */
export function accuracyTier(accuracy: number): "success" | "warning" | "error" {
  if (accuracy >= 0.7) return "success";
  if (accuracy >= 0.4) return "warning";
  return "error";
}
