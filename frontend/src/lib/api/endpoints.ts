/**
 * Thin named functions, one per backend route. Every one of these — and
 * nothing else — is how the app talks to the backend. No component calls
 * `apiFetch`/`fetch` directly; import from here (or from `@/lib/api`).
 *
 * Route source of truth: `backend/cmd/server/main.go`. See `types.ts` for
 * the DTO shapes and the deviations from `Documentation/openapi.yaml`.
 */

import { apiFetch, apiGet, apiPost } from "@/lib/api/client";
import { getAccessToken } from "@/lib/auth/token-store";
import type {
  AuthResponse,
  BatchExecuteRequest,
  BatchExecuteResponse,
  CloseSessionResponse,
  CompleteNodeRequest,
  EvidenceConfirmResponse,
  EvidenceConnectorRequest,
  EvidenceConnectorResponse,
  EvidenceUploadURLRequest,
  EvidenceUploadURLResponse,
  ErrorAnalysisResponse,
  ExecuteRequest,
  ExecuteResponse,
  GenerateRoadmapRequest,
  GetSessionResponse,
  HealthResponse,
  Job,
  LoginRequest,
  OAuthProvider,
  PracticeSession,
  ProblemSearchParams,
  ProblemSearchResponse,
  RegisterRequest,
  RequestHintRequest,
  RequestHintResponse,
  RoadmapActionResponse,
  RoadmapCurrentResponse,
  ScrapeProblemRequest,
  ScrapedProblem,
  SimilarProblemsResponse,
  SkipRequest,
  SkipResponse,
  StartSessionRequest,
  StartSessionResponse,
  SubmitRequest,
  SubmitResponse,
  TestOutRequest,
  TopicsResponse,
  User,
  UserProfileResponse,
  VerdictPollResponse,
} from "@/lib/api/types";

// ============================================================================
// Auth — backend/internal/handlers/auth_handler.go
// ============================================================================

/**
 * POST /api/auth/login — hits Next.js Route Handler
 */
export async function authLogin(req: LoginRequest): Promise<AuthResponse> {
  const res = await fetch("/api/auth/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(req),
    credentials: "include",
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: "Login failed" }));
    throw new Error(body.error || "Login failed");
  }
  return res.json();
}

/**
 * POST /api/auth/register — hits Next.js Route Handler
 */
export async function authRegister(req: RegisterRequest): Promise<AuthResponse> {
  const res = await fetch("/api/auth/register", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(req),
    credentials: "include",
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: "Registration failed" }));
    throw new Error(body.error || "Registration failed");
  }
  return res.json();
}

/**
 * NOT a fetch call — GET /auth/oauth/{provider}/redirect is a real browser
 * navigation (it 307s to the OAuth provider, which itself redirects back to
 * the backend's callback URL). Use as `window.location.href = oauthRedirectPath(...)`
 * or a plain `<a href>`, never through `apiFetch`.
 */
export function oauthRedirectPath(provider: OAuthProvider): string {
  return `/api/v1/auth/oauth/${provider}/redirect`;
}

/** GET /auth/me — returns the raw `User` row, not `UserProfileResponse`. */
export function getMe(): Promise<User> {
  return apiGet<User>("/auth/me");
}

/**
 * POST /api/auth/logout — this hits OUR Next.js Route Handler
 * (`src/app/api/auth/logout/route.ts`), not the backend directly. The
 * handler reads the httpOnly refresh cookie, calls the backend's
 * `POST /auth/logout` with it, clears the cookie, and denylists the access
 * token. Callers should also clear the in-memory access token afterward
 * (AuthProvider's `logout()` does this for you — prefer that over calling
 * this directly).
 */
export async function authLogout(): Promise<void> {
  const token = getAccessToken();
  await fetch("/api/auth/logout", {
    method: "POST",
    credentials: "include",
    headers: token ? { Authorization: `Bearer ${token}` } : undefined,
  });
}

// ============================================================================
// Roadmap — backend/internal/handlers/roadmap_handler.go
// ============================================================================

export function getRoadmap(): Promise<RoadmapCurrentResponse> {
  return apiGet<RoadmapCurrentResponse>("/roadmap");
}

/** Routed to the same handler as `getRoadmap()` — kept distinct for call-site clarity. */
export function getMyRoadmap(): Promise<RoadmapCurrentResponse> {
  return apiGet<RoadmapCurrentResponse>("/roadmap/me");
}

export function completeRoadmapNode(nodeId: string): Promise<RoadmapActionResponse> {
  const body: CompleteNodeRequest = { node_id: nodeId };
  return apiPost<RoadmapActionResponse>(`/roadmap/nodes/${nodeId}/complete`, body);
}

export function testOutRoadmapNode(nodeId: string): Promise<RoadmapActionResponse> {
  const body: TestOutRequest = { node_id: nodeId };
  return apiPost<RoadmapActionResponse>(`/roadmap/nodes/${nodeId}/test-out`, body);
}

/**
 * Response is a union: on success this is a full `RoadmapCurrentResponse`
 * (200, same shape as GET /roadmap); if the immediate re-fetch inside the
 * handler fails it falls back to `RoadmapActionResponse` instead. Narrow on
 * `"roadmap_id" in res`.
 */
export function generateRoadmap(
  req: GenerateRoadmapRequest = {},
): Promise<RoadmapCurrentResponse | RoadmapActionResponse> {
  return apiPost<RoadmapCurrentResponse | RoadmapActionResponse>("/roadmap/generate", req);
}

// ============================================================================
// Practice — backend/internal/handlers/practice_handler.go
// ============================================================================

export function startPracticeSession(req: StartSessionRequest): Promise<StartSessionResponse> {
  return apiPost<StartSessionResponse>("/practice/start", req);
}

/** Low-level. Prefer `submitSolution()` below, which does the full execute -> poll -> submit handshake. */
export function submitPracticeAnswer(req: SubmitRequest): Promise<SubmitResponse> {
  return apiPost<SubmitResponse>("/practice/submit", req);
}

export function skipPracticeProblem(req: SkipRequest): Promise<SkipResponse> {
  return apiPost<SkipResponse>("/practice/skip", req);
}

export function closePracticeSession(sessionId: string): Promise<CloseSessionResponse> {
  return apiPost<CloseSessionResponse>("/practice/close", { session_id: sessionId });
}

export function getPracticeSession(sessionId: string): Promise<GetSessionResponse> {
  return apiGet<GetSessionResponse>(`/practice/session/${sessionId}`);
}

/** POST /practice/{id}/hint — 202 Accepted, resolves async. See `resolveHint()` below. */
export function requestHint(
  sessionId: string,
  req: RequestHintRequest = {},
): Promise<RequestHintResponse> {
  return apiPost<RequestHintResponse>(`/practice/${sessionId}/hint`, req);
}

export function getErrorAnalysis(sessionId: string): Promise<ErrorAnalysisResponse> {
  return apiGet<ErrorAnalysisResponse>(`/practice/${sessionId}/error-analysis`);
}

export function getSimilarProblems(
  problemId: string,
  opts: { limit?: number; topic?: string } = {},
): Promise<SimilarProblemsResponse> {
  return apiGet<SimilarProblemsResponse>("/practice/similar", {
    query: { problem_id: problemId, limit: opts.limit, topic: opts.topic },
  });
}

export function getPracticeTopics(): Promise<TopicsResponse> {
  return apiGet<TopicsResponse>("/practice/topics");
}

// ============================================================================
// Execute — backend/internal/handlers/practice_handler.go (Judge0 bridge)
// ============================================================================

export function executeCode(req: ExecuteRequest): Promise<ExecuteResponse> {
  return apiPost<ExecuteResponse>("/execute", req);
}

export function getVerdict(token: string): Promise<VerdictPollResponse> {
  return apiGet<VerdictPollResponse>(`/execute/${token}`);
}

/** "Run all test cases" path. Independent of practice/submit — never feed this result into submitPracticeAnswer. */
export function executeBatch(req: BatchExecuteRequest): Promise<BatchExecuteResponse> {
  return apiPost<BatchExecuteResponse>("/execute/batch", req);
}

export interface PollVerdictOptions {
  /** ms between polls. Default 600ms per plan.md §3.1. */
  intervalMs?: number;
  /** total time budget before giving up. Default 30s per plan.md §3.1. */
  timeoutMs?: number;
  signal?: AbortSignal;
}

/**
 * Polls GET /execute/{token} until `is_done` (status_id > 2: anything past
 * "In Queue"/"Processing"), 600ms backoff, 30s ceiling — exactly the
 * contract in plan.md §3.1. Throws `Error("verdict poll timed out")` past
 * the ceiling.
 */
export async function pollVerdict(
  token: string,
  opts: PollVerdictOptions = {},
): Promise<VerdictPollResponse> {
  const intervalMs = opts.intervalMs ?? 600;
  const timeoutMs = opts.timeoutMs ?? 30_000;
  const deadline = Date.now() + timeoutMs;

  for (;;) {
    opts.signal?.throwIfAborted();
    const verdict = await getVerdict(token);
    if (verdict.is_done) return verdict;
    if (Date.now() >= deadline) {
      throw new Error("verdict poll timed out");
    }
    await new Promise((resolve) => setTimeout(resolve, intervalMs));
  }
}

export interface SubmitSolutionParams {
  sessionId: string;
  problemId?: string;
  languageId: number;
  sourceCode: string;
  customStdin?: string;
  timeTakenMs: number;
  pollOptions?: PollVerdictOptions;
}

/**
 * THE critical two-step handshake (plan.md §3.1 — the walkthrough doc got
 * this wrong; the handler is authoritative):
 *
 *   POST /execute -> { judge0_token }
 *   GET  /execute/{token}  (poll until is_done)
 *   POST /practice/submit { session_id, problem_id, judge0_token, time_taken_ms }
 *
 * This is the ONLY correct way to submit a solution in this app. Never call
 * `submitPracticeAnswer` with a token you didn't get from `executeCode`
 * immediately before, and never derive it from `executeBatch`.
 */
export async function submitSolution(params: SubmitSolutionParams): Promise<SubmitResponse> {
  const { judge0_token } = await executeCode({
    problem_id: params.problemId,
    language_id: params.languageId,
    source_code: params.sourceCode,
    custom_stdin: params.customStdin,
  });

  await pollVerdict(judge0_token, params.pollOptions);

  return submitPracticeAnswer({
    session_id: params.sessionId,
    problem_id: params.problemId,
    judge0_token,
    time_taken_ms: params.timeTakenMs,
  });
}

// ============================================================================
// Problems — backend/internal/handlers/practice_handler.go
// ============================================================================

export function searchProblems(params: ProblemSearchParams = {}): Promise<ProblemSearchResponse> {
  return apiGet<ProblemSearchResponse>("/problems/search", { query: { ...params } });
}

export function scrapeProblem(url: string): Promise<ScrapedProblem> {
  const body: ScrapeProblemRequest = { url };
  return apiPost<ScrapedProblem>("/problems/scrape", body);
}

// ============================================================================
// Jobs — backend/internal/jobs/handler.go
// ============================================================================

export function getJob(jobId: string): Promise<Job> {
  return apiGet<Job>(`/jobs/${jobId}`);
}

export interface ResolveHintOptions {
  /** Fallback poll interval if the SSE `hint.ready` event doesn't arrive first. Default 2s. */
  pollIntervalMs?: number;
  /** Give up after this long. Default 45s (LLM hints can be slow). */
  timeoutMs?: number;
  signal?: AbortSignal;
}

/**
 * Resolves a hint job by polling GET /jobs/{id} as a fallback path.
 * Prefer racing this against the `hint.ready` SSE event
 * (`useTransverseEvents` in `lib/realtime`) via `Promise.race` — this
 * function alone is the guaranteed-eventually-correct path if SSE drops.
 * Throws on `status === "failed"` or on timeout.
 */
export async function pollHintJob(jobId: string, opts: ResolveHintOptions = {}): Promise<Job> {
  const intervalMs = opts.pollIntervalMs ?? 2000;
  const timeoutMs = opts.timeoutMs ?? 45_000;
  const deadline = Date.now() + timeoutMs;

  for (;;) {
    opts.signal?.throwIfAborted();
    const job = await getJob(jobId);
    if (job.status === "done") return job;
    if (job.status === "failed") throw new Error(job.error ?? "hint job failed");
    if (Date.now() >= deadline) throw new Error("hint job poll timed out");
    await new Promise((resolve) => setTimeout(resolve, intervalMs));
  }
}

// ============================================================================
// User — backend/internal/handlers/user_handler.go
// ============================================================================

export function getUserProfile(): Promise<UserProfileResponse> {
  return apiGet<UserProfileResponse>("/user/profile");
}

export function getUserHistory(
  params: { limit?: number; offset?: number } = {},
): Promise<PracticeSession[]> {
  return apiGet<PracticeSession[]>("/user/history", { query: { ...params } });
}

// ============================================================================
// Evidence — backend/internal/handlers/evidence_handler.go
// Mounted by KEYSTONE per plan.md §9.1 — not on `main` as of this writing.
// ============================================================================

export function getEvidenceUploadUrl(
  req: EvidenceUploadURLRequest,
): Promise<EvidenceUploadURLResponse> {
  return apiPost<EvidenceUploadURLResponse>("/evidence/upload-url", req);
}

export function confirmEvidenceUpload(evidenceId: string): Promise<EvidenceConfirmResponse> {
  return apiPost<EvidenceConfirmResponse>(`/evidence/${evidenceId}/confirm`);
}

export function connectGithubEvidence(username: string): Promise<EvidenceConnectorResponse> {
  const body: EvidenceConnectorRequest = { username };
  return apiPost<EvidenceConnectorResponse>("/evidence/github", body);
}

export function connectLeetcodeEvidence(username: string): Promise<EvidenceConnectorResponse> {
  const body: EvidenceConnectorRequest = { username };
  return apiPost<EvidenceConnectorResponse>("/evidence/leetcode", body);
}

export function connectCodeforcesEvidence(handle: string): Promise<EvidenceConnectorResponse> {
  const body: EvidenceConnectorRequest = { handle };
  return apiPost<EvidenceConnectorResponse>("/evidence/codeforces", body);
}

// ============================================================================
// Health — backend/internal/handlers/health_handler.go
// GET /health lives OUTSIDE /api/v1 on the backend — see `skipPrefix` on
// `apiFetch`. Requires the extra `/health` rewrite in next.config.ts.
// ============================================================================

export async function getHealth(): Promise<HealthResponse> {
  return apiFetch<HealthResponse>("/health", { skipPrefix: true, skipAuthRefresh: true });
}
