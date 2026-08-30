"use client";

/**
 * `/onboarding/quiz` — the diagnostic quiz (plan.md §9.2, item 6 of the
 * route map). Reuses `/practice/*` as a short capped adaptive session
 * (`useQuizSession`, `src/components/quiz/use-quiz-session.ts`) across a
 * fixed seed-topic scope, one problem at a time, with a live theta +
 * confirmed/debunked-topic reading (`HypothesisMeter`) so the "hypothesis ->
 * verification" loop from the pitch is visible while it happens, not just
 * summarized afterward. On completion it closes the session and hands off
 * to `/onboarding/results`.
 */
import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { AlertTriangle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { AnswerSurface } from "@/components/practice/answer-surface";
import { Judge0Wait } from "@/components/practice/judge0-wait";
import { ProblemStatementPanel } from "@/components/practice/problem-statement-panel";
import { VerdictPanel } from "@/components/practice/verdict-panel";
import { LANGUAGES } from "@/mocks/fixtures/languages";
import { HypothesisMeter } from "@/components/quiz/hypothesis-meter";
import { QuizHeader } from "@/components/quiz/quiz-header";
import { QuizProgress } from "@/components/quiz/quiz-progress";
import { QUIZ_SEED_TOPICS } from "@/components/quiz/seed-topics";
import { useQuizSession } from "@/components/quiz/use-quiz-session";

export default function QuizPage() {
  const router = useRouter();
  const {
    state,
    start,
    setLanguage,
    setCode,
    submitAnswer,
    continueQuiz,
    skip,
    dismissError,
    cap,
  } = useQuizSession();

  useEffect(() => {
    void start();
  }, [start]);

  useEffect(() => {
    if (state.phase === "done" && state.sessionId) {
      router.push(`/onboarding/results?session=${state.sessionId}`);
    }
  }, [state.phase, state.sessionId, router]);

  const isBusy = state.phase === "starting" || state.phase === "skipping" || state.phase === "closing";

  return (
    <div className="min-h-full bg-tv-bg-page">
      <div className="mx-auto max-w-[1280px] px-6 py-10 md:px-12">
        <QuizHeader
          eyebrow="Onboarding · Step 2 of 3"
          title="Diagnostic Quiz"
          subtitle="A handful of rapid problems across core topics — enough for us to measure what you actually know before we build your roadmap. Skip anything that isn't a fair test."
          className="mb-6"
        />

        <QuizProgress questionCount={state.questionCount} cap={cap} className="mb-6" />

        {state.error && (
          <div
            role="alert"
            className="mb-6 flex items-start gap-2 rounded-tv-btn border border-tv-rose/30 bg-tv-rose/10 px-3 py-2 text-sm text-tv-rose"
          >
            <AlertTriangle className="mt-0.5 size-4 shrink-0" aria-hidden />
            <div className="flex-1">
              <p>{state.error}</p>
              {state.phase === "error" && (
                <Button variant="destructive" size="sm" className="mt-2" onClick={() => void start()}>
                  Retry
                </Button>
              )}
            </div>
            <button
              type="button"
              onClick={dismissError}
              aria-label="Dismiss"
              className="text-tv-rose/70 hover:text-tv-rose"
            >
              ×
            </button>
          </div>
        )}

        {state.phase === "starting" && !state.problem && (
          <div className="grid grid-cols-1 gap-6 lg:grid-cols-[1fr_320px]" aria-busy="true">
            <div className="space-y-3">
              <Skeleton className="h-8 w-2/3" />
              <Skeleton className="h-64 w-full" />
              <Skeleton className="h-48 w-full" />
            </div>
            <Skeleton className="h-64 w-full" />
          </div>
        )}

        {state.phase === "error" && !state.problem && (
          <div className="glass-panel rounded-tv-card p-8 text-center">
            <p className="mb-4 text-sm text-tv-text-body">
              We couldn&apos;t start the diagnostic quiz.
            </p>
            <Button onClick={() => void start()}>Try again</Button>
          </div>
        )}

        {state.problem && state.phase !== "done" && (
          <div className="grid grid-cols-1 gap-6 lg:grid-cols-[1fr_320px]">
            <div className="space-y-6">
              <ProblemStatementPanel problem={state.problem} />

              {state.phase === "judge0" && state.judge0StartedAt !== null ? (
                <Judge0Wait startedAt={state.judge0StartedAt} />
              ) : state.phase === "result" && state.lastResult ? (
                <div className="space-y-4">
                  <VerdictPanel result={state.lastResult} />
                  <Button onClick={continueQuiz} className="w-full sm:w-auto">
                    {state.questionCount >= cap || !state.lastResult.next_problem
                      ? "See my results"
                      : "Continue"}
                  </Button>
                </div>
              ) : (
                <>
                  <AnswerSurface
                    languages={LANGUAGES}
                    language={state.language}
                    onLanguageChange={setLanguage}
                    code={state.code}
                    onCodeChange={setCode}
                    disabled={isBusy}
                  />
                  <div className="flex flex-wrap gap-3">
                    <Button onClick={() => void submitAnswer()} disabled={isBusy || !state.code.trim()}>
                      Submit
                    </Button>
                    <Button variant="outline" onClick={() => void skip()} disabled={isBusy}>
                      Skip
                    </Button>
                  </div>
                </>
              )}
            </div>

            <div className="lg:sticky lg:top-6 lg:self-start">
              <HypothesisMeter
                theta={state.theta}
                topics={QUIZ_SEED_TOPICS}
                tally={state.topicTally}
              />
            </div>
          </div>
        )}

        {state.phase === "closing" && (
          <div className="glass-panel mt-6 flex items-center gap-3 rounded-tv-card p-6" role="status" aria-live="polite">
            <span
              aria-hidden
              className="size-5 animate-spin rounded-full border-2 border-tv-cyan/25 border-t-tv-cyan motion-reduce:animate-none"
            />
            <p className="font-mono text-sm text-tv-text-hi">Scoring your results…</p>
          </div>
        )}
      </div>
    </div>
  );
}
