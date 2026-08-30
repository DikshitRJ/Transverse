"use client";

/**
 * State machine backing `/practice` (the ongoing adaptive loop). Owns the
 * `POST /practice/start` -> submit-handshake -> `POST /practice/skip` ->
 * `POST /practice/close` lifecycle. See `use-quiz-session.ts` for the
 * sibling hook that drives the capped diagnostic quiz — kept separate
 * rather than over-abstracted into one "generic session" hook, because the
 * two flows genuinely diverge (open-ended vs capped, hints vs none, a
 * review step per verdict vs none for the quiz).
 *
 * Async handlers read from `stateRef` (a ref mirroring the latest state,
 * refreshed every render) rather than closing over the `state` from the
 * render that created the callback — `useCallback([])` gives each handler a
 * stable identity, so without the ref they'd otherwise see stale
 * `sessionId`/`problem`/`code` from whichever render first created them.
 */
import { useCallback, useEffect, useRef, useState } from "react";
import {
  closePracticeSession,
  skipPracticeProblem,
  startPracticeSession,
  submitSolution,
} from "@/lib/api/endpoints";
import { ApiError } from "@/lib/api/client";
import type {
  CloseSessionResponse,
  ProblemPayload,
  SessionScope,
  SubmitResponse,
} from "@/lib/api/types";
import { LANGUAGES, type LanguageMeta } from "@/mocks/fixtures/languages";
import { cacheCloseResult } from "./session-cache";

export type PracticePhase =
  | "setup"
  | "starting"
  | "active"
  | "judge0"
  | "result"
  | "skipping"
  | "exhausted"
  | "closing"
  | "closed"
  | "error";

export interface PracticeEngineState {
  phase: PracticePhase;
  sessionId: string | null;
  problem: ProblemPayload | null;
  language: LanguageMeta;
  code: string;
  theta: number;
  thetaStart: number;
  questionCount: number;
  sessionStatus: string | null;
  lastResult: SubmitResponse | null;
  judge0StartedAt: number | null;
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

const INITIAL_STATE: PracticeEngineState = {
  phase: "setup",
  sessionId: null,
  problem: null,
  language: DEFAULT_LANGUAGE,
  code: "",
  theta: 0,
  thetaStart: 0,
  questionCount: 0,
  sessionStatus: null,
  lastResult: null,
  judge0StartedAt: null,
  error: null,
};

export function usePracticeEngine() {
  const [state, setState] = useState<PracticeEngineState>(INITIAL_STATE);
  const stateRef = useRef(state);
  const problemLoadedAt = useRef<number>(Date.now());

  useEffect(() => {
    stateRef.current = state;
  }, [state]);

  const startSession = useCallback(async (mode: "ADAPTIVE" | "REGULAR", scope: SessionScope) => {
    setState((s) => ({ ...s, phase: "starting", error: null }));
    try {
      const res = await startPracticeSession({ mode, scope });
      problemLoadedAt.current = Date.now();
      setState((s) => ({
        ...s,
        phase: res.current_problem ? "active" : "exhausted",
        sessionId: res.session_id,
        problem: res.current_problem ?? null,
        theta: res.theta,
        thetaStart: res.theta,
        questionCount: 0,
        sessionStatus: res.status,
        code: templateFor(res.current_problem ?? null, s.language),
        lastResult: null,
      }));
    } catch (err) {
      setState((s) => ({
        ...s,
        phase: "error",
        error: err instanceof ApiError ? err.message : "Couldn't start a practice session.",
      }));
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
      setState((s) => ({
        ...s,
        phase: "result",
        lastResult: result,
        theta: result.theta_after,
        questionCount: result.question_count,
        sessionStatus: result.session_status,
        judge0StartedAt: null,
      }));
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

  const advanceToNext = useCallback(() => {
    setState((s) => {
      const next = s.lastResult?.next_problem ?? null;
      problemLoadedAt.current = Date.now();
      return {
        ...s,
        phase: next ? "active" : "exhausted",
        problem: next,
        code: templateFor(next, s.language),
        lastResult: null,
      };
    });
  }, []);

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
      problemLoadedAt.current = Date.now();
      setState((s) => ({
        ...s,
        phase: result.next_problem ? "active" : "exhausted",
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
  }, []);

  const endSession = useCallback(async (): Promise<CloseSessionResponse | null> => {
    const { sessionId } = stateRef.current;
    if (!sessionId) return null;

    setState((s) => ({ ...s, phase: "closing", error: null }));
    try {
      const result = await closePracticeSession(sessionId);
      cacheCloseResult(sessionId, result);
      setState((s) => ({ ...s, phase: "closed" }));
      return result;
    } catch (err) {
      setState((s) => ({
        ...s,
        phase: "active",
        error: err instanceof ApiError ? err.message : "Couldn't close the session.",
      }));
      return null;
    }
  }, []);

  const dismissError = useCallback(() => {
    setState((s) => ({ ...s, error: null }));
  }, []);

  return {
    state,
    startSession,
    setLanguage,
    setCode,
    submitAnswer,
    advanceToNext,
    skip,
    endSession,
    dismissError,
  };
}
