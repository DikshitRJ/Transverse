/**
 * In-memory practice session state machine backing POST /practice/start,
 * /submit, /skip, /close, and GET /practice/session/{id} — mirrors
 * `services.PracticeService` closely enough for realistic UI development
 * (theta nudges up on correct/down on incorrect, next-problem rotation,
 * per-topic close-out breakdown) without reimplementing the real IRT math.
 */
import type {
  CloseSessionResponse,
  GetSessionResponse,
  ProblemPayload,
  SessionResponse,
  SessionScope,
  SkipResponse,
  StartSessionResponse,
  SubmitResponse,
  TopicProgress,
  TopicsResponse,
  VerdictDetail,
} from "@/lib/api/types";
import { PROBLEMS } from "./problems";
import { getScenarioForToken } from "./verdicts";

interface MockSession {
  id: string;
  userId: string;
  mode: string;
  status: "ACTIVE" | "COMPLETED";
  scope: SessionScope;
  thetaStart: number;
  thetaCurrent: number;
  questionCount: number;
  currentProblem: ProblemPayload | null;
  seenProblemIds: Set<string>;
  responses: SessionResponse[];
}

const sessions = new Map<string, MockSession>();
let sessionCounter = 0;

function eligibleProblems(scope: SessionScope, seen: Set<string>): ProblemPayload[] {
  const pool = PROBLEMS.filter((p) => !seen.has(p.id));
  if (scope.topics && scope.topics.length > 0) {
    const scoped = pool.filter((p) => scope.topics.includes(p.topic));
    if (scoped.length > 0) return scoped;
  }
  return pool;
}

function pickNextProblem(session: MockSession): ProblemPayload | null {
  const pool = eligibleProblems(session.scope, session.seenProblemIds);
  if (pool.length === 0) return null;
  const idx = session.questionCount % pool.length;
  return pool[idx]!;
}

export function startSession(userId: string, mode: string, scope: SessionScope): StartSessionResponse {
  sessionCounter += 1;
  const id = `session-mock-${sessionCounter}-${Date.now()}`;
  const session: MockSession = {
    id,
    userId,
    mode,
    status: "ACTIVE",
    scope,
    thetaStart: 0.5,
    thetaCurrent: 0.5,
    questionCount: 0,
    currentProblem: null,
    seenProblemIds: new Set(),
    responses: [],
  };
  const first = pickNextProblem(session);
  session.currentProblem = first;
  if (first) session.seenProblemIds.add(first.id);
  sessions.set(id, session);

  return {
    session_id: id,
    mode,
    theta: session.thetaStart,
    current_problem: first ?? undefined,
    status: session.status,
    created_at: new Date().toISOString(),
  };
}

export function getSession(id: string): MockSession | undefined {
  return sessions.get(id);
}

export function submitAnswer(
  sessionId: string,
  judge0Token: string,
  timeTakenMs: number,
): SubmitResponse | undefined {
  const session = sessions.get(sessionId);
  if (!session) return undefined;

  const scenario = getScenarioForToken(judge0Token);
  const isCorrect = scenario?.statusId === 3;
  const thetaBefore = session.thetaCurrent;
  const delta = isCorrect ? 0.05 + Math.random() * 0.05 : -(0.03 + Math.random() * 0.04);
  session.thetaCurrent = Math.max(0, Math.min(1, thetaBefore + delta));
  session.questionCount += 1;

  const verdict: VerdictDetail = {
    status_id: scenario?.statusId ?? 3,
    status_desc: scenario?.statusDesc ?? "Accepted",
    time_ms: scenario?.timeMs ?? 40,
    memory_kb: scenario?.memoryKb ?? 8000,
    stderr: scenario?.stderr,
    compile_output: scenario?.compileOutput,
  };

  const solvedProblem = session.currentProblem;
  session.responses.push({
    problem_id: solvedProblem?.id ?? "",
    is_correct: isCorrect,
    skipped: false,
    judge0_status_id: verdict.status_id,
    judge0_status_desc: verdict.status_desc,
    execution_time_ms: verdict.time_ms,
    memory_kb: verdict.memory_kb,
    time_taken_ms: timeTakenMs,
    theta_before: thetaBefore,
    theta_after: session.thetaCurrent,
    question_count: session.questionCount,
    sc_score: isCorrect ? 0.8 : 0.2,
    difficulty_fit: 0.7,
    concept_similarity: 0.75,
    topic_progression: isCorrect ? 0.1 : 0,
    novelty_factor: 0.5,
    immediate_reinforce: isCorrect ? 1 : 0,
    platform_diversity: 0.4,
    carelessness_penalty: !isCorrect && verdict.status_id === 4 ? 0.15 : 0,
    theta_effective: session.thetaCurrent,
    momentum: isCorrect ? 0.2 : -0.1,
    submitted_at: new Date().toISOString(),
  });

  const next = pickNextProblem(session);
  session.currentProblem = next;
  if (next) session.seenProblemIds.add(next.id);

  return {
    is_correct: isCorrect,
    verdict,
    theta_before: thetaBefore,
    theta_after: session.thetaCurrent,
    next_problem: next ?? undefined,
    session_status: session.status,
    question_count: session.questionCount,
    carelessness_penalty: !isCorrect && verdict.status_id === 4 ? 0.15 : 0,
  };
}

export function skipProblem(sessionId: string, timeTakenMs: number): SkipResponse | undefined {
  const session = sessions.get(sessionId);
  if (!session) return undefined;

  const thetaBefore = session.thetaCurrent;
  session.thetaCurrent = Math.max(0, thetaBefore - 0.02);
  session.questionCount += 1;

  session.responses.push({
    problem_id: session.currentProblem?.id ?? "",
    is_correct: false,
    skipped: true,
    judge0_status_id: 0,
    judge0_status_desc: "Skipped",
    execution_time_ms: 0,
    memory_kb: 0,
    time_taken_ms: timeTakenMs,
    theta_before: thetaBefore,
    theta_after: session.thetaCurrent,
    question_count: session.questionCount,
    sc_score: 0,
    difficulty_fit: 0.5,
    concept_similarity: 0.5,
    topic_progression: 0,
    novelty_factor: 0.5,
    immediate_reinforce: 0,
    platform_diversity: 0.4,
    carelessness_penalty: 0,
    theta_effective: session.thetaCurrent,
    momentum: -0.05,
    submitted_at: new Date().toISOString(),
  });

  const next = pickNextProblem(session);
  session.currentProblem = next;
  if (next) session.seenProblemIds.add(next.id);

  return {
    skipped: true,
    theta_before: thetaBefore,
    theta_after: session.thetaCurrent,
    next_problem: next ?? undefined,
    question_count: session.questionCount,
  };
}

export function closeSession(sessionId: string): CloseSessionResponse | undefined {
  const session = sessions.get(sessionId);
  if (!session) return undefined;
  session.status = "COMPLETED";

  const byTopic = new Map<string, { attempts: number; correct: number }>();
  for (const r of session.responses) {
    const problem = PROBLEMS.find((p) => p.id === r.problem_id);
    const topic = problem?.topic ?? "unknown";
    const entry = byTopic.get(topic) ?? { attempts: 0, correct: 0 };
    entry.attempts += 1;
    if (r.is_correct) entry.correct += 1;
    byTopic.set(topic, entry);
  }

  const perTopicBreakdown: Record<string, TopicProgress> = {};
  for (const [topic, stats] of byTopic) {
    perTopicBreakdown[topic] = {
      topic,
      mastery_score: stats.attempts > 0 ? (stats.correct / stats.attempts) * 100 : 0,
      theta: 0,
      glicko_rating: 0,
      attempt_count: 0,
      correct_count: 0,
    };
  }

  const totalSolved = session.responses.filter((r) => r.is_correct).length;

  return {
    session_id: session.id,
    status: "CLOSED",
    theta_start: session.thetaStart,
    theta_final: session.thetaCurrent,
    mastery_score: session.responses.length > 0 ? (totalSolved / session.responses.length) * 100 : 0,
    accuracy: session.responses.length > 0 ? totalSolved / session.responses.length : 0,
    total_questions: session.responses.length,
    total_solved: totalSolved,
    per_topic_breakdown: perTopicBreakdown,
  };
}

export function toGetSessionResponse(session: MockSession): GetSessionResponse {
  return {
    session_id: session.id,
    user_id: session.userId,
    mode: session.mode,
    status: session.status,
    scope: session.scope,
    theta_start: session.thetaStart,
    theta_current: session.thetaCurrent,
    question_count: session.questionCount,
    current_problem: session.currentProblem ?? undefined,
    responses: session.responses,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  };
}

const TOPIC_ORDER = [
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
];

export function buildTopicsResponse(): TopicsResponse {
  return {
    topics: TOPIC_ORDER.map((topic, i) => ({
      topic,
      mastery_score: Math.max(0, 90 - i * 6),
      theta: Math.max(0, 0.8 - i * 0.05),
      glicko_rating: 1500 - i * 20,
      attempt_count: Math.max(0, 24 - i * 2),
      correct_count: Math.max(0, 18 - i * 2),
    })),
  };
}
