"use client";

/**
 * Drives `/onboarding/quiz` — the diagnostic (plan.md §9.2 / Decision B).
 * There's no dedicated quiz endpoint: this reuses `/practice/*` as a short,
 * capped adaptive session (`QUIZ_QUESTION_CAP`) scoped to `QUIZ_SEED_TOPICS`,
 * tracking a live per-topic tally client-side (the "hypothesis -> verification"
 * loop — see `hypothesis-meter.tsx`) and auto-closing once the cap or the
 * scoped pool is exhausted. See `use-practice-engine.ts` for why this isn't
 * shared with the ongoing `/practice` loop — the flows diverge enough
 * (capped vs open-ended, no hints, no manual "keep going" choice) that one
 * generic hook would need more branching than two small ones.
 */
import { useCallback, useEffect, useRef, useState } from "react";
import {
  closePracticeSession,
  skipPracticeProblem,
  startPracticeSession,
  submitSolution,
} from "@/lib/api/endpoints";
import { ApiError } from "@/lib/api/client";
import type { CloseSessionResponse, ProblemPayload, SubmitResponse } from "@/lib/api/types";
import { LANGUAGES, type LanguageMeta } from "@/mocks/fixtures/languages";
import { cacheCloseResult } from "@/components/practice/session-cache";
import { QUIZ_QUESTION_CAP, QUIZ_SEED_TOPICS } from "./seed-topics";

export type QuizPhase =
  | "starting"
  | "active"
  | "judge0"
  | "result"
  | "skipping"
  | "closing"
  | "done"
  | "error";

export interface TopicTally {
  attempts: number;
  correct: number;
}

export interface QuizState {
  phase: QuizPhase;
  sessionId: string | null;
  problem: ProblemPayload | null;
  language: LanguageMeta;
  code: string;
  theta: number;
  thetaStart: number;
  questionCount: number;
  topicTally: Record<string, TopicTally>;
  lastResult: SubmitResponse | null;
  judge0StartedAt: number | null;
  closeResult: CloseSessionResponse | null;
  error: string | null;
}

// A statically-known-safe fallback (mirrors LANGUAGES[0] exactly) rather than
// a non-null assertion — `LANGUAGES` is always non-empty in practice, but TS
// can't prove that from its `LanguageMeta[]` type.
const DEFAULT_LANGUAGE: LanguageMeta = LANGUAGES[0] ?? {
  key: "py",
  label: "Python 3",
  judge0Id: 71,
  monacoId: "python",
};

function templateFor(problem: ProblemPayload | null, language: LanguageMeta): string {
  return problem?.templates?.[language.key] ?? "";
}

const INITIAL_STATE: QuizState = {
  phase: "starting",
  sessionId: null,
  problem: null,
  language: DEFAULT_LANGUAGE,
  code: "",
  theta: 0,
  thetaStart: 0,
  questionCount: 0,
  topicTally: {},
  lastResult: null,
  judge0StartedAt: null,
  closeResult: null,
  error: null,
};

export function useQuizSession() {
  const [state, setState] = useState<QuizState>(INITIAL_STATE);
  const stateRef = useRef(state);
  const problemLoadedAt = useRef<number>(Date.now());
  const startInFlight = useRef(false);

  useEffect(() => {
    stateRef.current = state;
  }, [state]);

  const finish = useCallback(async (sessionId: string) => {
    setState((s) => ({ ...s, phase: "closing", error: null }));
    try {
      const result = await closePracticeSession(sessionId);
      cacheCloseResult(sessionId, result);
      setState((s) => ({ ...s, phase: "done", closeResult: result }));
    } catch (err) {
      setState((s) => ({
        ...s,
        phase: "error",
        error: err instanceof ApiError ? err.message : "Couldn't finalize your results.",
      }));
    }
  }, []);

  const start = useCallback(async () => {
    // Guards against true overlap (e.g. React StrictMode's dev-only
    // double-invoke of mount effects firing this twice back to back), not
    // against a later retry after a failed start — a "Retry" button calling
    // `start()` again must work, so this only blocks concurrent calls, and
    // always clears in `finally`.
    if (startInFlight.current) return;
    startInFlight.current = true;
    setState((s) => ({ ...s, phase: "starting", error: null }));
    try {
      const res = await startPracticeSession({
        mode: "ADAPTIVE",
        scope: { topics: QUIZ_SEED_TOPICS },
      });
      problemLoadedAt.current = Date.now();
      setState((s) => ({
        ...s,
        phase: "active",
        sessionId: res.session_id,
        problem: res.current_problem ?? null,
        theta: res.theta,
        thetaStart: res.theta,
        code: templateFor(res.current_problem ?? null, s.language),
      }));
    } catch (err) {
      setState((s) => ({
        ...s,
        phase: "error",
        error: err instanceof ApiError ? err.message : "Couldn't start the diagnostic quiz.",
      }));
    } finally {
      startInFlight.current = false;
    }
  }, []);

  const setLanguage = useCallback((language: LanguageMeta) => {
    setState((s) => ({ ...s, language, code: templateFor(s.problem, language) }));
  }, []);

  const setCode = useCallback((code: string) => {
    setState((s) => ({ ...s, code }));
  }, []);

  const submitAnswer = useCallback(async () => {
    const { sessionId, problem, language, code } = stateRef.current;
    if (!sessionId || !problem) return;

    setState((s) => ({ ...s, phase: "judge0", judge0StartedAt: Date.now(), error: null }));
    const timeTakenMs = Date.now() - problemLoadedAt.current;

    try {
      const result = await submitSolution({
        sessionId,
        problemId: problem.id,
        languageId: language.judge0Id,
        sourceCode: code,
        timeTakenMs,
      });
      setState((s) => {
        const prevTally = s.topicTally[problem.topic] ?? { attempts: 0, correct: 0 };
        const newCount = result.question_count > 0 ? result.question_count : s.questionCount + 1;
        return {
          ...s,
          phase: "result",
          lastResult: result,
          theta: result.theta_after,
          questionCount: newCount,
          judge0StartedAt: null,
          topicTally: {
            ...s.topicTally,
            [problem.topic]: {
              attempts: prevTally.attempts + 1,
              correct: prevTally.correct + (result.is_correct ? 1 : 0),
            },
          },
        };
      });
    } catch (err) {
      setState((s) => ({
        ...s,
        phase: "active",
        judge0StartedAt: null,
        error:
          err instanceof ApiError
            ? err.message
            : "Submission failed — check your connection and try again.",
      }));
    }
  }, []);

  const continueQuiz = useCallback(() => {
    const { sessionId, lastResult, questionCount } = stateRef.current;
    if (!sessionId) return;
    const next = lastResult?.next_problem ?? null;
    if (!next || questionCount >= QUIZ_QUESTION_CAP) {
      void finish(sessionId);
      return;
    }
    problemLoadedAt.current = Date.now();
    setState((s) => ({
      ...s,
      phase: "active",
      problem: next,
      code: templateFor(next, s.language),
      lastResult: null,
    }));
  }, [finish]);

  const skip = useCallback(async () => {
    const { sessionId, problem } = stateRef.current;
    if (!sessionId) return;

    setState((s) => ({ ...s, phase: "skipping", error: null }));
    const timeTakenMs = Date.now() - problemLoadedAt.current;

    try {
      const result = await skipPracticeProblem({
        session_id: sessionId,
        problem_id: problem?.id,
        time_taken_ms: timeTakenMs,
      });
      if (!result.next_problem || result.question_count >= QUIZ_QUESTION_CAP) {
        void finish(sessionId);
        return;
      }
      problemLoadedAt.current = Date.now();
      setState((s) => ({
        ...s,
        phase: "active",
        problem: result.next_problem ?? null,
        code: templateFor(result.next_problem ?? null, s.language),
        theta: result.theta_after,
        questionCount: result.question_count,
      }));
    } catch (err) {
      setState((s) => ({
        ...s,
        phase: "active",
        error: err instanceof ApiError ? err.message : "Couldn't skip that problem.",
      }));
    }
  }, [finish]);

  const skipQuiz = useCallback(async () => {
    const { sessionId } = stateRef.current;
    if (sessionId) {
      try {
        await finish(sessionId);
      } catch {
        if (typeof window !== "undefined") {
          window.location.href = `/onboarding/results?session=${sessionId}`;
        }
      }
    } else if (typeof window !== "undefined") {
      window.location.href = "/onboarding/results";
    }
  }, [finish]);

  const dismissError = useCallback(() => {
    setState((s) => ({ ...s, error: null }));
  }, []);

  return {
    state,
    start,
    setLanguage,
    setCode,
    submitAnswer,
    continueQuiz,
    skip,
    skipQuiz,
    dismissError,
    cap: QUIZ_QUESTION_CAP,
  };
}
