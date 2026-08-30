/**
 * MSW v2 request handlers for all 34 routes FOUNDRY owns (plan.md §7 —
 * the 29 `main.go` routes plus the 5 evidence routes KEYSTONE mounts).
 * Shared between the browser worker (`browser.ts`, client components) and
 * the Node server (`server.ts` / `instrumentation.ts`, RSC + Route
 * Handlers) — both must load this exact array so mock mode behaves
 * identically no matter where a request originates.
 *
 * Auth is enforced the same way the real `middleware.Auth` does: any
 * request to a protected route without an `Authorization: Bearer ...`
 * header gets the real `{"error": "..."}` 401 envelope. Get a token by
 * visiting `/signin` and clicking a provider — the mock OAuth redirect
 * short-circuits straight to `/auth/callback` with working mock tokens,
 * no real provider round-trip needed.
 */
import { http, HttpResponse, delay } from "msw";
import type {
  ApiErrorEnvelope,
  AuthTokenResponse,
  BatchExecuteRequest,
  CompleteNodeRequest,
  EvidenceConnectorRequest,
  EvidenceUploadURLRequest,
  ExecuteRequest,
  GenerateRoadmapRequest,
  HealthResponse,
  PracticeSession,
  RequestHintRequest,
  ScrapeProblemRequest,
  SessionResponse,
  SkipRequest,
  StartSessionRequest,
  SubmitRequest,
  TestOutRequest,
} from "@/lib/api/types";
import { PROBLEMS, getProblemById } from "./fixtures/problems";
import { roadmapState, findSubsection, buildRoadmap } from "./fixtures/roadmap";
import { MOCK_USER_ID, rawUser, userProfile } from "./fixtures/user";
import {
  createExecutionToken,
  pollExecutionToken,
  buildBatchResult,
} from "./fixtures/verdicts";
import {
  startSession,
  getSession,
  submitAnswer,
  skipProblem,
  closeSession,
  toGetSessionResponse,
  buildTopicsResponse,
} from "./fixtures/sessions";
import { createHintJob, getJobById } from "./fixtures/jobs";
import {
  createUploadEvidence,
  confirmEvidenceUpload,
  createConnectorEvidence,
} from "./fixtures/evidence";
import { emitMockEvent, subscribeMockEvents } from "./fixtures/event-bus";

const MOCK_ACCESS_TOKEN = "mock-access-token";

function errorBody(message: string): ApiErrorEnvelope {
  return { error: message };
}

function requireAuth(request: Request): string | null {
  const header = request.headers.get("authorization");
  if (!header || !header.startsWith("Bearer ") || header.length <= "Bearer ".length) {
    return null;
  }
  return "user-mock-001";
}

function unauthorized() {
  return HttpResponse.json(errorBody("missing or invalid authorization header"), { status: 401 });
}

function mockRefreshToken(): string {
  return `mock-refresh-${Date.now()}-${Math.random().toString(36).slice(2, 10)}`;
}

/**
 * `GET /user/history` returns `[]` in the real backend's real state today
 * (no sessions have actually been run against this mock user) — but
 * FOUNDATION.md explicitly permits PRISM to extend this one handler's
 * fixture rows so `/dashboard` and `/profile` have real data to chart
 * against. Deterministic (no `Math.random` in the data itself, only in
 * unrelated token helpers above) so the demo looks the same every run.
 * Shaped as a plausible history across the same roadmap topics
 * `fixtures/roadmap.ts`'s active section covers, with accuracy and θ
 * trending upward over time — a believable "user who's been improving."
 */
const HISTORY_TOPICS = ["foundations", "arrays-hashing", "two-pointers", "sliding-window", "stack-queues", "binary-search"];
const HISTORY_SESSION_COUNT = 16;
const DAY_MS = 24 * 60 * 60 * 1000;

function buildHistoryFixture(): PracticeSession[] {
  const now = Date.now();
  const sessions: PracticeSession[] = [];

  for (let i = 0; i < HISTORY_SESSION_COUNT; i++) {
    const progress = i / (HISTORY_SESSION_COUNT - 1);
    const daysAgo = (HISTORY_SESSION_COUNT - 1 - i) * 5 + (i % 3);
    const createdAt = new Date(now - daysAgo * DAY_MS);

    const topic = HISTORY_TOPICS[i % HISTORY_TOPICS.length]!;
    const secondTopic = HISTORY_TOPICS[(i + 1) % HISTORY_TOPICS.length]!;
    const topicProblems = PROBLEMS.filter((p) => p.topic === topic);

    const questionCount = 5 + (i % 4);
    const thetaStart = Number((0.28 + progress * 0.5).toFixed(3));
    const thetaCurrent = Number(Math.min(0.88, thetaStart + 0.03 + progress * 0.05).toFixed(3));
    const accuracyTarget = 0.4 + progress * 0.45;

    const responses: SessionResponse[] = Array.from({ length: questionCount }, (_, qi) => {
      const problem = topicProblems[qi % Math.max(1, topicProblems.length)];
      const isSkipped = i % 5 === 0 && qi === questionCount - 1;
      const isCorrect = !isSkipped && qi / questionCount < accuracyTarget;
      const submittedAt = new Date(createdAt.getTime() + qi * 4 * 60_000);
      const thetaBefore = Number((thetaStart + qi * 0.01).toFixed(3));
      const thetaAfter = Number((thetaStart + (qi + 1) * 0.01).toFixed(3));
      return {
        problem_id: problem?.id ?? "p-hero-000",
        is_correct: isCorrect,
        skipped: isSkipped,
        judge0_status_id: isSkipped ? 0 : isCorrect ? 3 : 4,
        judge0_status_desc: isSkipped ? "Skipped" : isCorrect ? "Accepted" : "Wrong Answer",
        execution_time_ms: 120 + qi * 17,
        memory_kb: 15_000 + qi * 512,
        time_taken_ms: 45_000 + qi * 8_000,
        theta_before: thetaBefore,
        theta_after: thetaAfter,
        question_count: qi + 1,
        sc_score: 0,
        difficulty_fit: 0,
        concept_similarity: 0,
        topic_progression: 0,
        novelty_factor: 0,
        immediate_reinforce: 0,
        platform_diversity: 0,
        carelessness_penalty: 0,
        theta_effective: thetaAfter,
        momentum: 0,
        submitted_at: submittedAt.toISOString(),
      };
    });

    const isMostRecent = i === HISTORY_SESSION_COUNT - 1;
    const status = isMostRecent ? "active" : i % 7 === 6 ? "abandoned" : "completed";

    sessions.push({
      id: `session-mock-${String(i).padStart(3, "0")}`,
      user_id: MOCK_USER_ID,
      mode: i % 3 === 0 ? "REGULAR" : "ADAPTIVE",
      scope: { topics: [topic, secondTopic] },
      theta_start: thetaStart,
      theta_current: thetaCurrent,
      current_problem_id: isMostRecent ? (topicProblems[0]?.id ?? null) : null,
      responses,
      question_count: questionCount,
      status,
      created_at: createdAt.toISOString(),
      updated_at: createdAt.toISOString(),
    });
  }

  return sessions.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime());
}

const HISTORY_FIXTURE = buildHistoryFixture();

export const handlers = [
  // ==========================================================================
  // Health — outside /api/v1
  // ==========================================================================
  http.get("*/health", async () => {
    await delay(20);
    const body: HealthResponse = { status: "ok", pool_total: 5, pool_idle: 4, pool_acquired: 1 };
    return HttpResponse.json(body);
  }),

  // ==========================================================================
  // Auth — public
  // ==========================================================================
  http.get("*/api/v1/auth/oauth/:provider/redirect", async () => {
    // Real backend 307s to the OAuth provider. Mock mode short-circuits
    // straight to the frontend callback with working tokens so sign-in is
    // exercisable with zero real OAuth app registration (plan.md §5.3 —
    // "agents build the sign-in UI against mocks").
    const url = new URL(
      `/auth/callback?access_token=${MOCK_ACCESS_TOKEN}&refresh_token=${mockRefreshToken()}&expires_in=3600`,
      "http://localhost",
    );
    return HttpResponse.redirect(`${url.pathname}${url.search}`, 307);
  }),

  http.get("*/api/v1/auth/oauth/:provider/callback", async () => {
    const url = new URL(
      `/auth/callback?access_token=${MOCK_ACCESS_TOKEN}&refresh_token=${mockRefreshToken()}&expires_in=3600`,
      "http://localhost",
    );
    return HttpResponse.redirect(`${url.pathname}${url.search}`, 307);
  }),

  http.post("*/api/v1/auth/refresh", async () => {
    await delay(150);
    const body: AuthTokenResponse = {
      access_token: MOCK_ACCESS_TOKEN,
      refresh_token: mockRefreshToken(),
      expires_in: 3600,
    };
    return HttpResponse.json(body);
  }),

  http.post("*/api/v1/auth/logout", async () => {
    await delay(80);
    return new HttpResponse(null, { status: 204 });
  }),

  // ==========================================================================
  // Auth — protected
  // ==========================================================================
  http.get("*/api/v1/auth/me", async ({ request }) => {
    if (!requireAuth(request)) return unauthorized();
    await delay(120);
    return HttpResponse.json(rawUser);
  }),

  // ==========================================================================
  // Roadmap
  // ==========================================================================
  http.get("*/api/v1/roadmap", async ({ request }) => {
    if (!requireAuth(request)) return unauthorized();
    await delay(150);
    return HttpResponse.json(roadmapState.current);
  }),

  http.get("*/api/v1/roadmap/me", async ({ request }) => {
    if (!requireAuth(request)) return unauthorized();
    await delay(150);
    return HttpResponse.json(roadmapState.current);
  }),

  http.post("*/api/v1/roadmap/nodes/:id/complete", async ({ request, params }) => {
    if (!requireAuth(request)) return unauthorized();
    await delay(200);
    const nodeId = String(params.id);
    const body = (await request.json().catch(() => ({}))) as Partial<CompleteNodeRequest>;
    void body;
    const sub = findSubsection(nodeId);
    if (sub) {
      sub.status = "mastered";
      sub.mastery_score = 1;
      const subsections = roadmapState.current.current_section?.subsections ?? [];
      const nextLocked = subsections.find((s) => s.status === "locked");
      if (nextLocked) {
        nextLocked.status = "unlocked";
        emitMockEvent({
          type: "node.unlocked",
          job_id: nextLocked.node_id,
          data: { node_id: nextLocked.node_id, topic_id: nextLocked.topic_id, title: nextLocked.title },
        });
        emitMockEvent({
          type: "roadmap.updated",
          job_id: roadmapState.current.roadmap_id,
          data: { roadmap_id: roadmapState.current.roadmap_id },
        });
      }
    }
    return HttpResponse.json({ success: true, message: "node successfully completed" });
  }),

  http.post("*/api/v1/roadmap/nodes/:id/test-out", async ({ request, params }) => {
    if (!requireAuth(request)) return unauthorized();
    await delay(200);
    const nodeId = String(params.id);
    const body = (await request.json().catch(() => ({}))) as Partial<TestOutRequest>;
    void body;
    const sub = findSubsection(nodeId);
    if (sub) {
      sub.status = "tested_out";
      sub.mastery_score = 1;
    }
    return HttpResponse.json({ success: true, message: "node successfully tested out" });
  }),

  http.post("*/api/v1/roadmap/generate", async ({ request }) => {
    if (!requireAuth(request)) return unauthorized();
    await delay(400);
    const body = (await request.json().catch(() => ({}))) as Partial<GenerateRoadmapRequest>;
    void body;
    roadmapState.current = buildRoadmap();
    emitMockEvent({
      type: "roadmap.updated",
      job_id: roadmapState.current.roadmap_id,
      data: { roadmap_id: roadmapState.current.roadmap_id },
    });
    return HttpResponse.json(roadmapState.current);
  }),

  // ==========================================================================
  // Practice
  // ==========================================================================
  http.post("*/api/v1/practice/start", async ({ request }) => {
    const userId = requireAuth(request);
    if (!userId) return unauthorized();
    await delay(200);
    const body = (await request.json()) as StartSessionRequest;
    const res = startSession(userId, body.mode || "ADAPTIVE", body.scope ?? { topics: [] });
    return HttpResponse.json(res);
  }),

  http.post("*/api/v1/practice/submit", async ({ request }) => {
    if (!requireAuth(request)) return unauthorized();
    await delay(250);
    const body = (await request.json()) as SubmitRequest;
    if (!body.session_id) {
      return HttpResponse.json(errorBody("session_id is required"), { status: 400 });
    }
    if (!body.judge0_token) {
      return HttpResponse.json(errorBody("judge0_token is required"), { status: 400 });
    }
    const res = submitAnswer(body.session_id, body.judge0_token, body.time_taken_ms);
    if (!res) return HttpResponse.json(errorBody("session not found"), { status: 404 });
    return HttpResponse.json(res);
  }),

  http.post("*/api/v1/practice/skip", async ({ request }) => {
    if (!requireAuth(request)) return unauthorized();
    await delay(180);
    const body = (await request.json()) as SkipRequest;
    if (!body.session_id) {
      return HttpResponse.json(errorBody("session_id is required"), { status: 400 });
    }
    const res = skipProblem(body.session_id, body.time_taken_ms);
    if (!res) return HttpResponse.json(errorBody("session not found"), { status: 404 });
    return HttpResponse.json(res);
  }),

  http.post("*/api/v1/practice/close", async ({ request }) => {
    if (!requireAuth(request)) return unauthorized();
    await delay(250);
    const body = (await request.json()) as { session_id: string };
    if (!body.session_id) {
      return HttpResponse.json(errorBody("session_id is required"), { status: 400 });
    }
    const res = closeSession(body.session_id);
    if (!res) return HttpResponse.json(errorBody("session not found"), { status: 404 });
    return HttpResponse.json(res);
  }),

  http.get("*/api/v1/practice/session/:id", async ({ request, params }) => {
    if (!requireAuth(request)) return unauthorized();
    await delay(150);
    const session = getSession(String(params.id));
    if (!session) {
      return HttpResponse.json(errorBody("session not found"), { status: 404 });
    }
    return HttpResponse.json(toGetSessionResponse(session));
  }),

  http.post("*/api/v1/practice/:id/hint", async ({ request, params }) => {
    const userId = requireAuth(request);
    if (!userId) return unauthorized();
    await delay(150);
    const body = (await request.json().catch(() => ({}))) as RequestHintRequest;
    const hintLevel = body.hint_level && body.hint_level > 0 ? body.hint_level : 1;
    const job = createHintJob(userId, hintLevel);
    void params.id;
    return HttpResponse.json({ job_id: job.id }, { status: 202 });
  }),

  http.get("*/api/v1/practice/:id/error-analysis", async ({ request, params }) => {
    if (!requireAuth(request)) return unauthorized();
    await delay(200);
    const session = getSession(String(params.id));
    const wrongCount = session?.responses.filter((r) => !r.is_correct && !r.skipped).length ?? 0;
    if (!session || wrongCount === 0) {
      return HttpResponse.json({
        status: "pending_or_not_found",
        message: "Error analysis results will appear here once the LLM job completes.",
      });
    }
    return HttpResponse.json({
      status: "done",
      message: "Pattern detected across recent misses.",
      weak_concepts: ["off-by-one boundary checks", "hash map key normalization"],
      recommendation:
        "Slow down on the boundary conditions before submitting — 2 of your last misses failed on an edge case, not the core algorithm.",
    });
  }),

  http.get("*/api/v1/practice/similar", async ({ request }) => {
    if (!requireAuth(request)) return unauthorized();
    await delay(150);
    const url = new URL(request.url);
    const problemId = url.searchParams.get("problem_id");
    if (!problemId) {
      return HttpResponse.json(errorBody("problem_id query parameter is required"), { status: 400 });
    }
    const limit = Number(url.searchParams.get("limit") ?? 5) || 5;
    const source = getProblemById(problemId);
    const similar = PROBLEMS.filter((p) => p.id !== problemId && p.topic === source?.topic).slice(
      0,
      limit,
    );
    return HttpResponse.json({ problem_id: problemId, similar_problems: similar });
  }),

  http.get("*/api/v1/practice/topics", async ({ request }) => {
    if (!requireAuth(request)) return unauthorized();
    await delay(150);
    return HttpResponse.json(buildTopicsResponse());
  }),

  // ==========================================================================
  // Execute
  // ==========================================================================
  http.post("*/api/v1/execute", async ({ request }) => {
    if (!requireAuth(request)) return unauthorized();
    await delay(120);
    const body = (await request.json()) as ExecuteRequest;
    if (!body.source_code) {
      return HttpResponse.json(errorBody("source_code cannot be empty"), { status: 400 });
    }
    if (!body.language_id || body.language_id <= 0) {
      return HttpResponse.json(errorBody("valid language_id is required"), { status: 400 });
    }
    const token = createExecutionToken();
    return HttpResponse.json({ judge0_token: token });
  }),

  http.get("*/api/v1/execute/:token", async ({ request, params }) => {
    if (!requireAuth(request)) return unauthorized();
    await delay(100);
    const verdict = pollExecutionToken(String(params.token));
    if (!verdict) {
      return HttpResponse.json(errorBody("unknown execution token"), { status: 404 });
    }
    return HttpResponse.json(verdict);
  }),

  http.post("*/api/v1/execute/batch", async ({ request }) => {
    if (!requireAuth(request)) return unauthorized();
    await delay(400);
    const body = (await request.json()) as BatchExecuteRequest;
    if (!body.source_code?.trim()) {
      return HttpResponse.json(errorBody("source_code cannot be empty"), { status: 400 });
    }
    if (!body.language_id || body.language_id <= 0) {
      return HttpResponse.json(errorBody("valid language_id is required"), { status: 400 });
    }
    let testCaseCount = body.test_cases?.length ?? 0;
    if (testCaseCount === 0 && body.problem_id) {
      testCaseCount = getProblemById(body.problem_id)?.test_cases?.length ?? 0;
    }
    if (testCaseCount === 0) {
      return HttpResponse.json(
        errorBody(
          "no test cases available for execution; provide test_cases in request or specify a problem_id with test cases",
        ),
        { status: 400 },
      );
    }
    return HttpResponse.json(buildBatchResult(testCaseCount));
  }),

  // ==========================================================================
  // Problems
  // ==========================================================================
  http.get("*/api/v1/problems/search", async ({ request }) => {
    if (!requireAuth(request)) return unauthorized();
    await delay(180);
    const url = new URL(request.url);
    const q = (url.searchParams.get("q") ?? "").toLowerCase().trim();
    const topic = url.searchParams.get("topic") ?? "";
    const source = url.searchParams.get("source") ?? "";
    const difficulty = url.searchParams.get("difficulty_label") ?? "";
    let limit = Number(url.searchParams.get("limit") ?? 20) || 20;
    if (limit > 100) limit = 100;
    const offset = Number(url.searchParams.get("offset") ?? 0) || 0;

    let results = PROBLEMS;
    if (q) results = results.filter((p) => p.name.toLowerCase().includes(q));
    if (topic) results = results.filter((p) => p.topic === topic);
    if (source) results = results.filter((p) => p.source === source);
    if (difficulty) results = results.filter((p) => p.difficulty_label === difficulty);

    const page = results.slice(offset, offset + limit);
    return HttpResponse.json({ total: results.length, problems: page });
  }),

  http.post("*/api/v1/problems/scrape", async ({ request }) => {
    if (!requireAuth(request)) return unauthorized();
    await delay(900);
    const body = (await request.json()) as ScrapeProblemRequest;
    if (!body.url?.trim()) {
      return HttpResponse.json(errorBody("url is required"), { status: 400 });
    }
    const fallback = PROBLEMS[0]!;
    return HttpResponse.json({
      title: `Scraped: ${new URL(body.url).pathname.split("/").filter(Boolean).pop() ?? "problem"}`,
      source: body.url.includes("codeforces") ? "codeforces" : "leetcode",
      url: body.url,
      statement: fallback.statement,
      test_cases: fallback.test_cases,
      tags: fallback.tags,
      difficulty: fallback.difficulty_label,
      templates: fallback.templates,
    });
  }),

  // ==========================================================================
  // Realtime — GET /events/stream
  // ==========================================================================
  http.get("*/api/v1/events/stream", ({ request }) => {
    if (!requireAuth(request)) return unauthorized();

    const encoder = new TextEncoder();
    let unsubscribe: (() => void) | null = null;
    let heartbeat: ReturnType<typeof setInterval> | null = null;

    const stream = new ReadableStream({
      start(controller) {
        controller.enqueue(encoder.encode("event: connected\ndata: {}\n\n"));
        unsubscribe = subscribeMockEvents((event) => {
          controller.enqueue(encoder.encode(`data: ${JSON.stringify(event)}\n\n`));
        });
        heartbeat = setInterval(() => {
          controller.enqueue(encoder.encode(": keepalive\n\n"));
        }, 15_000);
      },
      cancel() {
        unsubscribe?.();
        if (heartbeat) clearInterval(heartbeat);
      },
    });

    return new HttpResponse(stream, {
      headers: {
        "Content-Type": "text/event-stream",
        "Cache-Control": "no-cache",
        Connection: "keep-alive",
      },
    });
  }),

  // ==========================================================================
  // Jobs
  // ==========================================================================
  http.get("*/api/v1/jobs/:id", async ({ request, params }) => {
    if (!requireAuth(request)) return unauthorized();
    await delay(80);
    const job = getJobById(String(params.id));
    if (!job) return HttpResponse.json(errorBody("job not found"), { status: 404 });
    return HttpResponse.json(job);
  }),

  // ==========================================================================
  // User
  // ==========================================================================
  http.get("*/api/v1/user/profile", async ({ request }) => {
    if (!requireAuth(request)) return unauthorized();
    await delay(150);
    return HttpResponse.json(userProfile);
  }),

  http.get("*/api/v1/user/history", async ({ request }) => {
    if (!requireAuth(request)) return unauthorized();
    await delay(150);
    const url = new URL(request.url);
    const limit = Number(url.searchParams.get("limit") ?? HISTORY_FIXTURE.length) || HISTORY_FIXTURE.length;
    const offset = Number(url.searchParams.get("offset") ?? 0) || 0;
    return HttpResponse.json(HISTORY_FIXTURE.slice(offset, offset + limit));
  }),

  // ==========================================================================
  // Evidence — routed by KEYSTONE per plan.md §9.1
  // ==========================================================================
  http.post("*/api/v1/evidence/upload-url", async ({ request }) => {
    const userId = requireAuth(request);
    if (!userId) return unauthorized();
    await delay(200);
    const body = (await request.json()) as EvidenceUploadURLRequest;
    if (body.kind !== "resume" && body.kind !== "codebase") {
      return HttpResponse.json(errorBody("invalid kind for upload"), { status: 400 });
    }
    const { evidenceId, uploadUrl } = createUploadEvidence(userId, body.kind);
    return HttpResponse.json({ evidence_id: evidenceId, upload_url: uploadUrl });
  }),

  http.post("*/api/v1/evidence/:id/confirm", async ({ request, params }) => {
    if (!requireAuth(request)) return unauthorized();
    await delay(150);
    const entry = confirmEvidenceUpload(String(params.id));
    if (!entry) return HttpResponse.json(errorBody("evidence not found"), { status: 500 });
    return HttpResponse.json({ status: "processing_queued" }, { status: 202 });
  }),

  http.post("*/api/v1/evidence/github", async ({ request }) => {
    const userId = requireAuth(request);
    if (!userId) return unauthorized();
    await delay(200);
    const body = (await request.json()) as EvidenceConnectorRequest;
    const id = createConnectorEvidence(userId, "github");
    void body;
    return HttpResponse.json({ evidence_id: id, status: "processing_queued" }, { status: 202 });
  }),

  http.post("*/api/v1/evidence/leetcode", async ({ request }) => {
    const userId = requireAuth(request);
    if (!userId) return unauthorized();
    await delay(200);
    const body = (await request.json()) as EvidenceConnectorRequest;
    const id = createConnectorEvidence(userId, "leetcode");
    void body;
    return HttpResponse.json({ evidence_id: id, status: "processing_queued" }, { status: 202 });
  }),

  http.post("*/api/v1/evidence/codeforces", async ({ request }) => {
    const userId = requireAuth(request);
    if (!userId) return unauthorized();
    await delay(200);
    const body = (await request.json()) as EvidenceConnectorRequest;
    const id = createConnectorEvidence(userId, "codeforces");
    void body;
    return HttpResponse.json({ evidence_id: id, status: "processing_queued" }, { status: 202 });
  }),
];
