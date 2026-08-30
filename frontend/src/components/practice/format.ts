/**
 * Small formatting helpers shared by the quiz + practice surfaces (PULSE).
 * Kept dependency-free and pure so both `components/quiz/**` and
 * `components/practice/**` can import from here without cross-owning state.
 */

/** "arrays-hashing" -> "Arrays Hashing" */
export function topicLabel(topicId: string): string {
  if (!topicId) return "General";
  return topicId
    .split("-")
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

/** 184320 -> "3m 4s", 4200 -> "4.2s", 340 -> "340ms" */
export function formatDuration(ms: number): string {
  if (!Number.isFinite(ms) || ms < 0) return "—";
  if (ms < 1000) return `${Math.round(ms)}ms`;
  const totalSeconds = ms / 1000;
  if (totalSeconds < 60) return `${totalSeconds.toFixed(1)}s`;
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = Math.round(totalSeconds % 60);
  return `${minutes}m ${seconds}s`;
}

/** 8192 -> "8.0 MB", 512 -> "512 KB" */
export function formatMemory(kb: number): string {
  if (!Number.isFinite(kb) || kb <= 0) return "—";
  if (kb < 1024) return `${Math.round(kb)} KB`;
  return `${(kb / 1024).toFixed(1)} MB`;
}

/** 0.6234 -> "62%". For a 0-1 fraction (e.g. `accuracy`) — NOT `mastery_score`, see `formatMasteryScore`. */
export function formatPercent(fraction: number): string {
  if (!Number.isFinite(fraction)) return "—";
  return `${Math.round(Math.max(0, Math.min(1, fraction)) * 100)}%`;
}

/**
 * 62.3 -> "62%". For every `mastery_score` field in the API — `TopicProgress`
 * (both `GET /practice/topics` and `CloseSessionResponse.per_topic_breakdown`)
 * and `CloseSessionResponse.mastery_score` itself — which is already on a
 * 0-100 scale, NOT a 0-1 fraction (verified against
 * `backend/internal/services/practice_analytics.go:10`,
 * `CalculateMasteryScore`: baseline theta -> 0, 2800+ -> 100). Passing one of
 * those fields through `formatPercent` instead would silently 100x it.
 */
export function formatMasteryScore(score: number): string {
  if (!Number.isFinite(score)) return "—";
  return `${Math.round(Math.max(0, Math.min(100, score)))}%`;
}

/** Clamp + format a theta-like value to 2 decimals, e.g. 0.5432 -> "0.54" */
export function formatTheta(theta: number): string {
  if (!Number.isFinite(theta)) return "0.00";
  return theta.toFixed(2);
}

const DIFFICULTY_BADGE: Record<string, "success" | "warning" | "error"> = {
  easy: "success",
  medium: "warning",
  hard: "error",
  expert: "error",
};

export function difficultyBadgeVariant(label: string): "success" | "warning" | "error" {
  return DIFFICULTY_BADGE[label.toLowerCase()] ?? "warning";
}
