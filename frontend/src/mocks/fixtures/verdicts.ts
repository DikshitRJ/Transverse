/**
 * Scripted Judge0 verdict sequences for the mock POST /execute ->
 * GET /execute/{token} handshake. Every call to POST /execute advances a
 * counter and cycles through 4 realistic scenarios so Wave-1 UI (FORGE's
 * IDE, PULSE's practice loop) can exercise every verdict state without
 * needing to hand-craft broken code: accepted, wrong answer, TLE, compile
 * error. `status_id`/`status_desc` follow real Judge0 CE conventions.
 *
 * Poll progression is call-count-based (queue -> processing -> final) so it
 * works regardless of the caller's actual polling interval — 2 polls to
 * settle, matching a fast but not-instant Judge0 run.
 */
import type { TestCaseResult, VerdictPollResponse } from "@/lib/api/types";

export interface Scenario {
  statusId: number;
  statusDesc: string;
  stdout?: string;
  stderr?: string;
  compileOutput?: string;
  message?: string;
  timeMs: number;
  memoryKb: number;
}

const SCENARIOS: Scenario[] = [
  { statusId: 3, statusDesc: "Accepted", stdout: "ok", timeMs: 42, memoryKb: 8192 },
  {
    statusId: 4,
    statusDesc: "Wrong Answer",
    stdout: "wrong output",
    timeMs: 38,
    memoryKb: 7900,
  },
  {
    statusId: 5,
    statusDesc: "Time Limit Exceeded",
    timeMs: 2000,
    memoryKb: 16400,
  },
  {
    statusId: 6,
    statusDesc: "Compilation Error",
    compileOutput: "error: expected ';' before '}' token",
    timeMs: 0,
    memoryKb: 0,
  },
];

interface TokenState {
  polls: number;
  scenario: Scenario;
}

const tokenStore = new Map<string, TokenState>();
let executeCounter = 0;

export function createExecutionToken(): string {
  const scenario = SCENARIOS[executeCounter % SCENARIOS.length]!;
  executeCounter += 1;
  const token = `mock-judge0-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
  tokenStore.set(token, { polls: 0, scenario });
  return token;
}

export function pollExecutionToken(token: string): VerdictPollResponse | undefined {
  const state = tokenStore.get(token);
  if (!state) return undefined;

  state.polls += 1;

  // Queue -> Processing -> final, settling on the 3rd poll.
  if (state.polls === 1) {
    return {
      token,
      status_id: 1,
      status_desc: "In Queue",
      time_ms: 0,
      memory_kb: 0,
      is_done: false,
    };
  }
  if (state.polls === 2) {
    return {
      token,
      status_id: 2,
      status_desc: "Processing",
      time_ms: 0,
      memory_kb: 0,
      is_done: false,
    };
  }

  const s = state.scenario;
  return {
    token,
    status_id: s.statusId,
    status_desc: s.statusDesc,
    time_ms: s.timeMs,
    memory_kb: s.memoryKb,
    stdout: s.stdout,
    stderr: s.stderr,
    compile_output: s.compileOutput,
    message: s.message,
    is_done: true,
  };
}

/**
 * Looks up the final scenario for a token without advancing its poll
 * counter — used by `sessions.ts:submitAnswer` to decide `is_correct`
 * once the client has already polled the token to completion (mirrors the
 * real `PracticeService.Submit`, which re-fetches the verdict server-side
 * rather than trusting the client's poll history).
 */
export function getScenarioForToken(token: string): Scenario | undefined {
  return tokenStore.get(token)?.scenario;
}

/** For POST /execute/batch — synchronous, no token/poll cycle. */
export function buildBatchResult(testCaseCount: number): {
  all_passed: boolean;
  passed_count: number;
  total_count: number;
  overall_status: string;
  overall_status_id: number;
  max_time_ms: number;
  max_memory_kb: number;
  test_cases: TestCaseResult[];
} {
  const testCases: TestCaseResult[] = Array.from({ length: testCaseCount }, (_, i) => {
    const passed = i !== testCaseCount - 1 || testCaseCount === 1; // last case fails when there's more than one, for a realistic "almost passing" run
    return {
      index: i,
      input: `case ${i + 1} input`,
      expected_output: `case ${i + 1} expected`,
      actual_stdout: passed ? `case ${i + 1} expected` : `case ${i + 1} unexpected`,
      status_id: passed ? 3 : 4,
      status_desc: passed ? "Accepted" : "Wrong Answer",
      time_ms: 20 + i * 4,
      memory_kb: 7800 + i * 50,
      passed,
    };
  });

  const passedCount = testCases.filter((t) => t.passed).length;
  const allPassed = passedCount === testCaseCount;

  return {
    all_passed: allPassed,
    passed_count: passedCount,
    total_count: testCaseCount,
    overall_status: allPassed ? "Accepted" : "Wrong Answer",
    overall_status_id: allPassed ? 3 : 4,
    max_time_ms: Math.max(...testCases.map((t) => t.time_ms)),
    max_memory_kb: Math.max(...testCases.map((t) => t.memory_kb)),
    test_cases: testCases,
  };
}
