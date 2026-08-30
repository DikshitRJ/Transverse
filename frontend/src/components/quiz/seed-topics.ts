/**
 * The diagnostic quiz's fixed topic scope (plan.md §9.2 — "a short capped
 * session of ~8-10 questions across seed topics"). There is no curriculum
 * endpoint to pull this from, so it's a hand-picked spread across the
 * topics that actually have problems in the fixture pool
 * (`mocks/fixtures/problems.ts` — both the hand-authored "hero" problems and
 * the generated fillers), wide enough that an 8-10 question session rarely
 * repeats the same topic twice.
 */
export const QUIZ_SEED_TOPICS: string[] = [
  "arrays-hashing",
  "two-pointers",
  "sliding-window",
  "stack-queues",
  "binary-search",
  "linked-list",
  "trees",
  "dynamic-programming",
  "graphs",
];

/** Diagnostic quiz length cap (plan.md §9.2 — "~8-10 questions"). */
export const QUIZ_QUESTION_CAP = 8;

/**
 * `mastery_score` threshold separating a "confirmed" from a "debunked"
 * hypothesis on close-out. `mastery_score` is a 0-100 scale, not a 0-1
 * fraction (verified against `backend/internal/services/practice_analytics.go:10`,
 * `CalculateMasteryScore`) — this threshold is in that same 0-100 space.
 */
export const HYPOTHESIS_CONFIRM_THRESHOLD = 60;
