"use client";

/**
 * `/practice` — the ongoing adaptive loop (plan.md route 14). Unlike
 * `/onboarding/quiz` this is open-ended: pick a mode + topic scope,
 * `POST /practice/start`, then repeat submit-handshake / skip / hint for as
 * long as the learner wants before explicitly ending the session, which
 * routes to `/practice/session/[id]` for the summary.
 */
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { AlertTriangle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Footer } from "@/components/shell/footer";
import { PageContainer } from "@/components/shell/page-container";
import { TopNav } from "@/components/shell/top-nav";
import { AnswerSurface } from "@/components/practice/answer-surface";
import { HintPanel } from "@/components/practice/hint-panel";
import { Judge0Wait } from "@/components/practice/judge0-wait";
import { ProblemStatementPanel } from "@/components/practice/problem-statement-panel";
import { SimilarProblems } from "@/components/practice/similar-problems";
import { ThetaGauge } from "@/components/practice/theta-gauge";
import { TopicScopePicker } from "@/components/practice/topic-scope-picker";
import { usePracticeEngine } from "@/components/practice/use-practice-engine";
import { useHint } from "@/components/practice/use-hint";
import { VerdictPanel } from "@/components/practice/verdict-panel";
import { LANGUAGES } from "@/mocks/fixtures/languages";

export default function PracticePage() {
  const router = useRouter();
  const {
    state,
    startSession,
    setLanguage,
    setCode,
    submitAnswer,
    advanceToNext,
    skip,
    endSession,
    dismissError,
  } = usePracticeEngine();
  const hint = useHint(state.sessionId);

  const [mode, setMode] = useState<"ADAPTIVE" | "REGULAR">("ADAPTIVE");
  const [scopeTopics, setScopeTopics] = useState<string[]>([]);

  useEffect(() => {
    if (state.phase === "closed" && state.sessionId) {
      router.push(`/practice/session/${state.sessionId}`);
    }
  }, [state.phase, state.sessionId, router]);

  const toggleTopic = (topic: string) => {
    setScopeTopics((prev) =>
      prev.includes(topic) ? prev.filter((t) => t !== topic) : [...prev, topic],
    );
  };

  const isBusy = state.phase === "starting" || state.phase === "skipping" || state.phase === "closing";
  const sessionActive = state.sessionId !== null && state.phase !== "closed";

  return (
    <div className="flex min-h-full flex-col bg-tv-bg-page">
      <TopNav />
      <PageContainer className="flex-1">
        <div className="mb-6 flex flex-wrap items-center justify-between gap-4">
          <h1 className="font-display text-h1 font-bold tracking-[-1px] text-tv-text-hi uppercase">
            Practice
          </h1>
          {sessionActive && (
            <div className="flex items-center gap-4">
              <span className="font-mono text-xs text-tv-text-body">
                Question {state.questionCount}
              </span>
              <ThetaGauge theta={state.theta} size="sm" className="w-40" />
              <Button
                variant="outline"
                size="sm"
                onClick={() => void endSession()}
                disabled={state.phase === "closing"}
              >
                End session
              </Button>
            </div>
          )}
        </div>

        {state.error && (
          <div
            role="alert"
            className="mb-6 flex items-start gap-2 rounded-tv-btn border border-tv-rose/30 bg-tv-rose/10 px-3 py-2 text-sm text-tv-rose"
          >
            <AlertTriangle className="mt-0.5 size-4 shrink-0" aria-hidden />
            <p className="flex-1">{state.error}</p>
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

        {!sessionActive && state.phase !== "closing" ? (
          <Card className="glow-card-cyan">
            <CardHeader>
              <CardTitle className="font-display text-h4 uppercase text-tv-text-hi">
                Start a session
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-5">
              <div>
                <p className="mb-2 font-mono text-xs text-tv-text-body uppercase">Mode</p>
                <div className="flex gap-2">
                  {(["ADAPTIVE", "REGULAR"] as const).map((m) => (
                    <button
                      key={m}
                      type="button"
                      onClick={() => setMode(m)}
                      aria-pressed={mode === m}
                      className={`rounded-tv-btn border px-3 py-1.5 font-mono text-xs uppercase transition-colors ${
                        mode === m
                          ? "border-tv-cyan bg-tv-cyan/10 text-tv-cyan glow-text-cyan"
                          : "border-tv-border text-tv-text-body hover:border-tv-border-cyan hover:text-tv-text-hi"
                      }`}
                    >
                      {m === "ADAPTIVE" ? "Adaptive" : "Regular"}
                    </button>
                  ))}
                </div>
              </div>

              <div>
                <p className="mb-2 font-mono text-xs text-tv-text-body uppercase">
                  Topic scope (leave empty for all topics)
                </p>
                <TopicScopePicker selected={scopeTopics} onToggle={toggleTopic} />
              </div>

              <Button
                onClick={() => void startSession(mode, { topics: scopeTopics })}
                disabled={state.phase === "starting"}
              >
                {state.phase === "starting" ? "Starting…" : "Start practicing"}
              </Button>
            </CardContent>
          </Card>
        ) : state.phase === "closing" ? (
          <div className="glass-panel flex items-center gap-3 rounded-tv-card p-6" role="status" aria-live="polite">
            <span
              aria-hidden
              className="size-5 animate-spin rounded-full border-2 border-tv-cyan/25 border-t-tv-cyan motion-reduce:animate-none"
            />
            <p className="font-mono text-sm text-tv-text-hi">Closing session…</p>
          </div>
        ) : (
          <div className="grid grid-cols-1 gap-6 lg:grid-cols-[1fr_320px]">
            <div className="space-y-6">
              {state.phase === "starting" && !state.problem ? (
                <div className="space-y-3" aria-busy="true">
                  <Skeleton className="h-8 w-2/3" />
                  <Skeleton className="h-64 w-full" />
                </div>
              ) : (
                <ProblemStatementPanel problem={state.problem} />
              )}

              {state.phase === "judge0" && state.judge0StartedAt !== null ? (
                <Judge0Wait startedAt={state.judge0StartedAt} />
              ) : state.phase === "result" && state.lastResult ? (
                <div className="space-y-4">
                  <VerdictPanel result={state.lastResult} />
                  {state.lastResult.next_problem ? (
                    <Button onClick={advanceToNext}>Next problem</Button>
                  ) : (
                    <p className="font-mono text-xs text-tv-text-body">
                      No more problems in this scope — end the session to see your summary.
                    </p>
                  )}
                </div>
              ) : state.phase === "exhausted" ? (
                <div className="glass-panel rounded-tv-card p-6 text-center">
                  <p className="mb-3 text-sm text-tv-text-body">
                    You&apos;ve worked through every problem in this scope.
                  </p>
                  <Button onClick={() => void endSession()}>End session</Button>
                </div>
              ) : state.problem ? (
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
                  <HintPanel hint={hint} />
                </>
              ) : null}
            </div>

            <div className="space-y-6 lg:sticky lg:top-6 lg:self-start">
              {state.problem && <SimilarProblems problemId={state.problem.id} />}
            </div>
          </div>
        )}
      </PageContainer>
      <Footer />
    </div>
  );
}
