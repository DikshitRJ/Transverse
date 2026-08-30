/**
 * Mock user identity + profile. `rawUser` mirrors what GET /auth/me
 * actually returns (`models.User`, raw `dna` JSONB) — `userProfile`
 * mirrors GET /user/profile (`models.UserProfileResponse`, decoded
 * `LearningDNA`). These are deliberately different shapes, see the
 * comments on both types in `lib/api/types.ts`.
 */
import type { LearningDNA, User, UserProfileResponse } from "@/lib/api/types";

export const MOCK_USER_ID = "user-mock-001";

export const learningDNA: LearningDNA = {
  avg_accuracy: 0.71,
  avg_time_taken_ms: 214_000,
  avg_solve_velocity: 0.62,
  carelessness_index: 0.18,
  peak_performance_hour: 21,
  avg_session_length: 34.5,
  total_sessions: 27,
  total_problems_solved: 96,
  topic_bias: {
    "arrays-hashing": 0.82,
    "two-pointers": 0.64,
    "sliding-window": 0.31,
    "stack-queues": 0.12,
    "dynamic-programming": 0.08,
    graphs: 0.05,
  },
  preferred_platform: "leetcode",
  streak_record: 11,
};

export const userProfile: UserProfileResponse = {
  id: MOCK_USER_ID,
  username: "byte_learner",
  email: "learner@example.com",
  theta: 0.84,
  glicko_rating: 1340,
  glicko_rd: 68,
  dna: learningDNA,
  created_at: "2026-06-01T12:00:00Z",
};

export const rawUser: User = {
  id: MOCK_USER_ID,
  username: "byte_learner",
  email: "learner@example.com",
  theta: 0.84,
  glicko_rating: 1340,
  glicko_rd: 68,
  glicko_vol: 0.06,
  dna: learningDNA,
  created_at: "2026-06-01T12:00:00Z",
  updated_at: "2026-08-29T09:30:00Z",
};
