/**
 * Filter option lists for `/problems`. `TOPIC_OPTIONS` mirrors the real
 * curriculum topic ids used across the app (roadmap fixture, `practice/
 * topics` mock — see `mocks/fixtures/sessions.ts`'s `TOPIC_ORDER`);
 * `SOURCE_OPTIONS`/`DIFFICULTY_OPTIONS` mirror the mock problem bank's
 * `SOURCES`/`DIFFICULTIES` (`mocks/fixtures/problems.ts`). These are UI
 * convenience lists for the filter `<Select>`s, not part of the typed API
 * contract — `ProblemSearchParams.topic`/`.source`/`.difficulty_label` are
 * plain strings either way.
 */
export const TOPIC_OPTIONS = [
  "foundations",
  "arrays-hashing",
  "two-pointers",
  "sliding-window",
  "stack-queues",
  "binary-search",
  "sorting-searching",
  "linked-list",
  "trees",
  "tries",
  "heaps-priority-queues",
  "backtracking",
  "graphs",
  "dynamic-programming",
  "greedy",
  "bit-manipulation",
] as const;

export const SOURCE_OPTIONS = ["leetcode", "codeforces", "atcoder", "cses"] as const;

export const DIFFICULTY_OPTIONS = ["easy", "medium", "hard", "expert"] as const;
