/**
 * Transverse API types — hand-transcribed from the Go backend, NOT from
 * `Documentation/openapi.yaml` (which has drifted — see plan.md §0.2).
 *
 * Each block below carries a comment naming the exact Go file it mirrors.
 * If you change a Go DTO, update the matching block here in the same PR —
 * this file (plus the MSW fixtures built from it) is the contract every
 * other frontend surface codes against.
 *
 * Known deliberate deviations from `openapi.yaml` (verified against the Go
 * source as of this writing):
 *  - `TestCaseResult.index` (NOT `test_case_index`)
 *  - `ProblemPayload` has `tags[]`, `subtopic`, `avg_time_ms`, `contest_id`
 *    and has NO `glicko_rating` or `status` field
 *  - `SubmitResponse` also carries `session_status`, `question_count`,
 *    `carelessness_penalty`
 *  - Submitting is a two-step handshake: POST /execute -> judge0_token,
 *    then POST /practice/submit with that token. See client helper in
 *    `execute.ts`.
 *  - The error envelope actually returned by every handler is the flat
 *    `{ "error": "message string" }` (see `handlers/helpers.go:writeError`),
 *    NOT the nested `{ error: { code, message } }` shape assumed by the
 *    original plan. `client.ts` normalises to `ApiError` from this real
 *    shape.
 */

// ============================================================================
// Shared primitives — backend/internal/models/db_models.go (TestCase)
// ============================================================================

/** Mirrors `models.TestCase` (db_models.go). */
export interface TestCase {
  input: string;
  output: string;
  explanation?: string;
  is_hidden?: boolean;
}

// ============================================================================
// backend/internal/models/dto.go
// ============================================================================

/** Mirrors `models.ProblemPayload` — sanitized problem, no internal psychometrics. */
export interface ProblemPayload {
  id: string;
  source: string;
  name: string;
  url: string;
  slug: string;
  contest_id?: string;
  tags: string[];
  topic: string;
  subtopic: string;
  difficulty_label: string;
  solve_rate: number;
  avg_time_ms: number;
  statement?: string;
  test_cases?: TestCase[];
  /** language name (e.g. "python3", "cpp17") -> starter source */
  templates?: Record<string, string>;
}

/** Mirrors `models.SessionScope` (db_models.go) — used in StartSessionRequest. */
export interface SessionScope {
  topics: string[];
  subtopics?: string[];
  sources?: string[];
  /** [min, max] Glicko rating */
  difficulty_range?: [number, number];
}

/** Mirrors `models.StartSessionRequest`. */
export interface StartSessionRequest {
  mode: "ADAPTIVE" | "REGULAR";
  scope: SessionScope;
}

/** Mirrors `models.StartSessionResponse`. */
export interface StartSessionResponse {
  session_id: string;
  mode: string;
  theta: number;
  current_problem?: ProblemPayload;
  status: string;
  created_at: string;
}

/** Mirrors `models.VerdictDetail`. */
export interface VerdictDetail {
  status_id: number;
  status_desc: string;
  time_ms: number;
  memory_kb: number;
  stderr?: string;
  compile_output?: string;
  message?: string;
}

/**
 * Body for POST /practice/submit. Mirrors the handler-local `submitPayload`
 * (practice_handler.go) — NOT `models.SubmitRequest`, which is unused by the
 * route as written (a documentation/implementation drift noted for KEYSTONE).
 */
export interface SubmitRequest {
  session_id: string;
  problem_id?: string;
  judge0_token: string;
  time_taken_ms: number;
}

/** Mirrors `models.SubmitResponse`. */
export interface SubmitResponse {
  is_correct: boolean;
  verdict: VerdictDetail;
  theta_before: number;
  theta_after: number;
  next_problem?: ProblemPayload;
  session_status: string;
  question_count: number;
  carelessness_penalty?: number;
}

/** Mirrors the handler-local `skipPayload` (practice_handler.go) for POST /practice/skip. */
export interface SkipRequest {
  session_id: string;
  problem_id?: string;
  time_taken_ms: number;
}

/** Mirrors `models.SkipResponse`. */
export interface SkipResponse {
  skipped: boolean;
  theta_before: number;
  theta_after: number;
  next_problem?: ProblemPayload;
  question_count: number;
}

/** Body for POST /practice/close. Mirrors handler-local `closePayload`. */
export interface CloseSessionRequest {
  session_id: string;
}

/** Mirrors `models.TopicProgress`. */
export interface TopicProgress {
  topic: string;
  mastery_score: number;
  theta: number;
  glicko_rating: number;
  attempt_count: number;
  correct_count: number;
}

/**
 * Mirrors `models.CloseSessionResponse`. NOTE: the handler only ever
 * populates `TopicProgress.topic` and `.mastery_score` on this response's
 * per-topic breakdown (see `PracticeHandler.CloseSession`) — `theta`,
 * `glicko_rating`, `attempt_count`, `correct_count` will be zero values here
 * even though the type allows them; use GET /practice/topics for the full
 * picture.
 */
export interface CloseSessionResponse {
  session_id: string;
  status: string;
  theta_start: number;
  theta_final: number;
  mastery_score: number;
  accuracy: number;
  total_questions: number;
  total_solved: number;
  per_topic_breakdown: Record<string, TopicProgress>;
}

/** Mirrors `models.SessionResponse` (db_models.go) — one logged attempt within a session. */
export interface SessionResponse {
  problem_id: string;
  is_correct: boolean;
  skipped: boolean;
  judge0_status_id: number;
  judge0_status_desc: string;
  execution_time_ms: number;
  memory_kb: number;
  time_taken_ms: number;
  theta_before: number;
  theta_after: number;
  question_count: number;
  sc_score: number;
  difficulty_fit: number;
  concept_similarity: number;
  topic_progression: number;
  novelty_factor: number;
  immediate_reinforce: number;
  platform_diversity: number;
  carelessness_penalty: number;
  theta_effective: number;
  momentum: number;
  submitted_at: string;
}

/** Mirrors `models.GetSessionResponse`. */
export interface GetSessionResponse {
  session_id: string;
  user_id: string;
  mode: string;
  status: string;
  scope: SessionScope;
  theta_start: number;
  theta_current: number;
  question_count: number;
  current_problem?: ProblemPayload;
  responses: SessionResponse[];
  created_at: string;
  updated_at: string;
}

/** Mirrors `models.TopicsResponse`. */
export interface TopicsResponse {
  topics: TopicProgress[];
}

/** Mirrors `models.LearningDNA` (db_models.go). */
export interface LearningDNA {
  avg_accuracy: number;
  avg_time_taken_ms: number;
  avg_solve_velocity: number;
  carelessness_index: number;
  peak_performance_hour: number;
  avg_session_length: number;
  total_sessions: number;
  total_problems_solved: number;
  topic_bias: Record<string, number>;
  preferred_platform: string;
  streak_record: number;
}

/** Mirrors `models.UserProfileResponse`. */
export interface UserProfileResponse {
  id: string;
  username: string;
  email: string;
  theta: number;
  glicko_rating: number;
  glicko_rd: number;
  dna: LearningDNA;
  created_at: string;
}

/** Mirrors `models.SimilarProblemsResponse` (query params only, no request body — see endpoints.ts). */
export interface SimilarProblemsResponse {
  problem_id: string;
  similar_problems: ProblemPayload[];
}

/** Mirrors `models.ExecuteRequest`. */
export interface ExecuteRequest {
  problem_id?: string;
  language_id: number;
  source_code: string;
  custom_stdin?: string;
}

/** Mirrors `models.ExecuteResponse`. */
export interface ExecuteResponse {
  judge0_token: string;
}

/** Mirrors `models.VerdictPollResponse`. `status_id > 2` means done (1 = In Queue, 2 = Processing). */
export interface VerdictPollResponse {
  token: string;
  status_id: number;
  status_desc: string;
  time_ms: number;
  memory_kb: number;
  stdout?: string;
  stderr?: string;
  compile_output?: string;
  message?: string;
  is_done: boolean;
}

/** Mirrors `models.TestCaseResult`. Field is `index`, NOT `test_case_index`. */
export interface TestCaseResult {
  index: number;
  input?: string;
  expected_output?: string;
  actual_stdout?: string;
  stderr?: string;
  compile_output?: string;
  status_id: number;
  status_desc: string;
  time_ms: number;
  memory_kb: number;
  passed: boolean;
}

/** Mirrors `models.BatchExecuteRequest`. This is the "Run all test cases" path — its result never feeds /practice/submit. */
export interface BatchExecuteRequest {
  problem_id?: string;
  language_id: number;
  source_code: string;
  test_cases?: TestCase[];
}

/** Mirrors `models.BatchExecuteResponse`. */
export interface BatchExecuteResponse {
  all_passed: boolean;
  passed_count: number;
  total_count: number;
  overall_status: string;
  overall_status_id: number;
  max_time_ms: number;
  max_memory_kb: number;
  test_cases: TestCaseResult[];
}

/**
 * Mirrors `models.ProblemSearchRequest` conceptually, but GET /problems/search
 * takes these as URL query params — note the query-string key is `q`, not
 * `query` (see `PracticeHandler.SearchProblems`, which reads `r.URL.Query().Get("q")`).
 */
export interface ProblemSearchParams {
  q?: string;
  topic?: string;
  source?: string;
  difficulty_label?: string;
  limit?: number;
  offset?: number;
}

/** Mirrors `models.ProblemSearchResponse`. */
export interface ProblemSearchResponse {
  total: number;
  problems: ProblemPayload[];
}

/** Body for POST /problems/scrape. Mirrors handler-local `scrapePayload`. */
export interface ScrapeProblemRequest {
  url: string;
}

/** Mirrors `scraper.ScrapedProblem` (internal/scraper/scraper.go) — the response of POST /problems/scrape. */
export interface ScrapedProblem {
  title: string;
  source: string;
  url: string;
  statement: string;
  time_limit?: string;
  memory_limit?: string;
  input_specification?: string;
  output_specification?: string;
  test_cases: TestCase[];
  tags?: string[];
  difficulty?: string;
  templates?: Record<string, string>;
}

// ============================================================================
// backend/internal/models/roadmap.go
// ============================================================================

export type RoadmapSource = "llm_generated" | "curated";

export type RoadmapStatus = "active" | "completed" | "abandoned";

export type NodeStatus =
  | "locked"
  | "unlocked"
  | "in_progress"
  | "mastered"
  | "tested_out";

/** Mirrors `models.UnlockRule`. */
export interface UnlockRule {
  type: "no_prerequisite" | "mastery_threshold" | "phase_complete" | "quiz_pass";
  topic_id?: string;
  min_rating?: number;
  phase_id?: string;
}

/** Mirrors `models.Tutorial`. */
export interface Tutorial {
  id: string;
  source: string;
  source_url: string;
  title: string;
  topic_id?: string;
  topic_tags: string[];
  type: "article" | "video" | "interactive" | "playlist";
  difficulty: "beginner" | "intermediate" | "advanced";
  estimated_minutes: number;
  summary: string;
  license_note?: string;
  thumbnail_url?: string;
  status: "COMPLETED" | "UNREAD";
}

/** Mirrors `models.RoadmapSubsection`. */
export interface RoadmapSubsection {
  node_id: string;
  topic_id: string;
  title: string;
  sequence: number;
  status: NodeStatus;
  user_rating: number;
  target_rating: number;
  mastery_score: number;
  tutorials: Tutorial[];
  questions: ProblemPayload[];
}

/** Mirrors `models.RoadmapSection`. */
export interface RoadmapSection {
  phase_id: string;
  sequence: number;
  title: string;
  status: "ACTIVE" | "COMPLETED" | "LOCKED";
  progress_percentage: number;
  subsections: RoadmapSubsection[];
}

/** Mirrors `models.UpcomingSectionPreview`. */
export interface UpcomingSectionPreview {
  sequence: number;
  title: string;
  status: "LOCKED";
}

/**
 * Mirrors `models.RoadmapCurrentResponse` — the payload for GET /roadmap and
 * GET /roadmap/me (both routed to the same handler). Only `current_section`
 * is fully populated; `upcoming_sections` are metadata-only previews.
 */
export interface RoadmapCurrentResponse {
  roadmap_id: string;
  user_id: string;
  user_rating: number;
  target_role: string;
  status: RoadmapStatus;
  total_sections: number;
  current_section: RoadmapSection | null;
  upcoming_sections: UpcomingSectionPreview[];
}

/** Mirrors `models.CompleteNodeRequest`. Body for POST /roadmap/nodes/{id}/complete. */
export interface CompleteNodeRequest {
  node_id: string;
}

/** Mirrors `models.TestOutRequest`. Body for POST /roadmap/nodes/{id}/test-out. */
export interface TestOutRequest {
  node_id: string;
}

/** Generic `{success, message}` ack shape used by CompleteNode / TestOutNode. */
export interface RoadmapActionResponse {
  success: boolean;
  message: string;
}

/**
 * Body for POST /roadmap/generate. Mirrors the handler-local
 * `generateRoadmapPayload` (roadmap_handler.go) — not in models/roadmap.go.
 * All fields optional; target_role defaults server-side to
 * "Software Engineer - DSA & Problem Solving".
 */
export interface GenerateRoadmapRequest {
  target_role?: string;
  confirmed_hypotheses?: string[];
  debunked_hypotheses?: string[];
}

// ============================================================================
// backend/internal/models/evidence.go
// ============================================================================

export type EvidenceKind =
  | "github"
  | "leetcode"
  | "codeforces"
  | "resume"
  | "codebase";

export type EvidenceStatus =
  | "pending"
  | "fetching"
  | "processing"
  | "done"
  | "failed"
  | "purged";

/**
 * Request/response DTOs below mirror `backend/internal/handlers/evidence_handler.go`
 * (handler-local structs — evidence.go itself only defines the DB-facing
 * `EvidenceSource`/`EvidenceExtract` entities, which are never serialized to
 * the client directly). These 5 routes are mounted by KEYSTONE per plan.md
 * §9.1 — they do not exist on `main` yet as of this writing.
 */

/** Body for POST /evidence/upload-url. Only `resume` and `codebase` are valid `kind`s here. */
export interface EvidenceUploadURLRequest {
  kind: Extract<EvidenceKind, "resume" | "codebase">;
  filename: string;
}

export interface EvidenceUploadURLResponse {
  evidence_id: string;
  upload_url: string;
}

/** Response for POST /evidence/{id}/confirm (202 Accepted). */
export interface EvidenceConfirmResponse {
  status: "processing_queued";
}

/**
 * Body for POST /evidence/github and POST /evidence/leetcode (`username`),
 * and POST /evidence/codeforces (`handle`). Send only the field the target
 * connector expects; the other is ignored server-side.
 */
export interface EvidenceConnectorRequest {
  username?: string;
  handle?: string;
}

/** Response for the three connector routes (202 Accepted). */
export interface EvidenceConnectorResponse {
  evidence_id: string;
  status: "processing_queued";
}

// ============================================================================
// backend/internal/models/db_models.go (User — full shape returned by GET /auth/me)
// ============================================================================

/**
 * Mirrors `models.User` verbatim — this is what GET /auth/me returns
 * (`writeJSON(w, http.StatusOK, user)` in auth_handler.go), NOT
 * `UserProfileResponse`. Note `dna` here is the *raw, undecoded* JSONB blob
 * (`json.RawMessage` server-side) — parse it defensively, or prefer
 * GET /user/profile (`UserProfileResponse`) which already decodes it into
 * `LearningDNA` with `DefaultDNA()` fallback. Also note the field is
 * `glicko_vol` here, not `glicko_volatility` (that name is used on `Problem`
 * instead) — the two structs are inconsistent in the Go source itself.
 */
export interface User {
  id: string;
  username: string;
  email: string;
  theta: number;
  glicko_rating: number;
  glicko_rd: number;
  glicko_vol: number;
  /** Raw JSONB — may be `null`, `{}`, or a serialized LearningDNA. Parse defensively. */
  dna: unknown;
  created_at: string;
  updated_at: string;
}

/**
 * Mirrors `models.PracticeSession` (db_models.go) — the raw shape returned
 * by GET /user/history (`[]models.PracticeSession`, unlike GET
 * /practice/session/{id} which returns the decoded `GetSessionResponse`).
 * `scope` and `responses` are the RAW JSONB columns here (not yet decoded
 * server-side) — parse them as `SessionScope` / `SessionResponse[]`
 * defensively, same caveat as `User.dna` above.
 */
export interface PracticeSession {
  id: string;
  user_id: string;
  mode: string;
  scope: unknown;
  theta_start: number;
  theta_current: number;
  current_problem_id: string | null;
  responses: unknown;
  question_count: number;
  status: string;
  created_at: string;
  updated_at: string;
}

// ============================================================================
// backend/internal/handlers/auth_handler.go
// ============================================================================

/** Response of the OAuth token issuance path (`issueTokens`), used by POST /auth/refresh. */
export interface AuthTokenResponse {
  access_token: string;
  refresh_token: string;
  /** seconds until access_token expiry */
  expires_in: number;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface RegisterRequest {
  email: string;
  password: string;
  username?: string;
}

export interface AuthResponse {
  access_token: string;
  refresh_token?: string;
  expires_in: number;
  user?: User;
}

export interface RefreshRequest {
  refresh_token: string;
}

export interface LogoutRequest {
  refresh_token?: string;
}

export type OAuthProvider = "github";

// ============================================================================
// backend/internal/jobs/queue.go — GET /jobs/{id}, and the job.completed /
// job.failed SSE event payloads
// ============================================================================

export type JobStatus = "queued" | "running" | "done" | "failed";

/** Mirrors `jobs.Job`. `output` is only present once `status === "done"`. */
export interface Job {
  id: string;
  user_id: string;
  job_type: string;
  status: JobStatus;
  input_ref?: unknown;
  /** Raw JSON payload produced by the worker (e.g. a hint's text). */
  output?: unknown;
  error?: string;
  created_at: string;
  started_at?: string;
  completed_at?: string;
}

// ============================================================================
// backend/internal/handlers/practice_handler.go — async hint request
// ============================================================================

/** Body for POST /practice/{id}/hint. Defaults to hint_level=1 server-side if omitted/<=0. */
export interface RequestHintRequest {
  hint_level?: number;
}

/** Response of POST /practice/{id}/hint (202 Accepted). */
export interface RequestHintResponse {
  job_id: string;
}

/**
 * Response of GET /practice/{id}/error-analysis. The service currently
 * stubs this out (`PracticeService.GetErrorAnalysis`) pending the LLM
 * worker table — treat as an opaque, evolving bag of fields. The only
 * fields guaranteed today are `status` and `message`.
 */
export interface ErrorAnalysisResponse {
  status: string;
  message?: string;
  [key: string]: unknown;
}

// ============================================================================
// GET /health — backend/internal/handlers/health_handler.go
// ============================================================================

export interface HealthResponse {
  status: "ok";
  pool_total: number;
  pool_idle: number;
  pool_acquired: number;
}

export interface HealthErrorResponse {
  status: "error";
  error: string;
}

// ============================================================================
// Realtime — backend/internal/realtime/handler.go + jobs/queue.go PublishEvent
// ============================================================================

/**
 * Every SSE frame from GET /events/stream is a plain `data: <json>\n\n` line
 * (the backend never sets a named SSE `event:` field except the literal
 * initial `event: connected\ndata: {}\n\n` handshake frame) — so a native
 * `EventSource.onmessage` (or a manual fetch/ReadableStream reader, which is
 * what this app uses because EventSource cannot send the required
 * `Authorization` header) always receives this envelope shape and must
 * discriminate on `.type` itself. See `jobs.redisQueue.PublishEvent`.
 *
 * As of this writing the backend only ever publishes `job.completed` and
 * `job.failed` (from `jobs/worker.go`). `node.unlocked`, `roadmap.updated`,
 * and `hint.ready` are part of the design (plan.md §2 "Cross-cutting") but
 * are not yet emitted anywhere server-side — the MSW mock stream fires them
 * on a script so Wave-1 UI can be built against them now; wire the real
 * ones up once KEYSTONE/backend adds the publishers.
 */
export type TransverseEventType =
  | "job.completed"
  | "job.failed"
  | "node.unlocked"
  | "roadmap.updated"
  | "hint.ready";

export interface TransverseEvent<T = unknown> {
  type: TransverseEventType;
  job_id: string;
  data: T;
}

/** `data` shape for a `hint.ready` event — the job's decoded `output`. */
export interface HintReadyEventData {
  hint_level: number;
  hint_text: string;
}

/** `data` shape for a `node.unlocked` event. */
export interface NodeUnlockedEventData {
  node_id: string;
  topic_id: string;
  title: string;
}

/** `data` shape for a `roadmap.updated` event. */
export interface RoadmapUpdatedEventData {
  roadmap_id: string;
}

// ============================================================================
// API error envelope — backend/internal/handlers/helpers.go
// ============================================================================

/**
 * The ACTUAL shape written by every handler on error (`writeError`):
 * `{"error": "message string"}` — flat, not `{error:{code,message}}`.
 * `client.ts` normalises this into `ApiError` (message + status + optional
 * raw body) so callers never touch this envelope directly.
 */
export interface ApiErrorEnvelope {
  error: string;
}
