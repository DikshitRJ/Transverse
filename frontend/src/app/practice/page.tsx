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
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
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

  const isBusy = state.phase === "starting" || state.phase === "skipping";
  const sessionActive = state.sessionId !== null && state.phase !== "closed";

  return (
    <div className="flex h-screen min-h-0 flex-col bg-tv-bg">
      <TopNav />

      {/* Main Workspace Area */}
      <div className="min-h-0 flex-1 flex flex-col">
        {state.error && (
          <div
            role="alert"
            className="m-3 flex items-start gap-2 rounded-tv-btn border border-tv-rose/30 bg-tv-rose/10 px-3 py-2 text-sm text-tv-rose"
          >
            <AlertTriangle className="mt-0.5 size-4 shrink-0" aria-hidden />
            <p className="flex-1 font-mono text-xs">{state.error}</p>
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

        {!sessionActive ? (
          <PageContainer className="flex-1 py-8">
            <div className="mb-6">
              <h1 className="font-display text-h1 font-bold tracking-[-1px] text-tv-text-hi uppercase">
                Practice
              </h1>
              <p className="font-mono text-xs text-tv-text-body mt-1">
                Continuous adaptive problem solving calibrated to your ability parameter (&theta;).
              </p>
            </div>

            <Card className="glow-card-cyan max-w-2xl">
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
                  className="w-full sm:w-auto"
                >
                  {state.phase === "starting" ? "Starting…" : "Start practicing"}
                </Button>
              </CardContent>
            </Card>
          </PageContainer>
        ) : (
          <div className="flex h-full min-h-0 flex-col">
            {/* LeetCode Header Toolbar */}
            <div className="flex shrink-0 flex-wrap items-center justify-between gap-3 border-b border-tv-border bg-tv-surface px-4 py-2.5">
              <div className="flex items-center gap-3">
                <span className="font-mono text-xs font-semibold text-tv-cyan">
                  Question {state.questionCount}
                </span>
                {state.problem && (
                  <>
                    <span className="text-tv-border">|</span>
                    <span className="font-display text-sm font-bold text-tv-text-hi truncate max-w-xs sm:max-w-md">
                      {state.problem.name}
                    </span>
                  </>
                )}
              </div>

              <div className="flex items-center gap-3">
                <ThetaGauge theta={state.theta} size="sm" className="w-32 sm:w-40" />
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => void endSession()}
                  disabled={isBusy}
                  className="font-mono text-xs"
                >
                  End Session
                </Button>
              </div>
            </div>

            {/* 2-Column Split Workspace */}
            <div className="grid min-h-0 flex-1 grid-cols-1 divide-y divide-tv-border lg:grid-cols-2 lg:divide-y-0 lg:divide-x">
              {/* Left Column: Problem Tabs (Description, Hints, Similar) */}
              <div className="flex h-full min-h-0 flex-col bg-tv-bg overflow-hidden">
                <Tabs defaultValue="description" className="flex h-full min-h-0 flex-col">
                  <div className="border-b border-tv-border bg-tv-surface px-3 py-1.5 shrink-0">
                    <TabsList className="h-8 bg-tv-surface-deep">
                      <TabsTrigger value="description" className="text-xs font-mono">
                        Description
                      </TabsTrigger>
                      <TabsTrigger value="hints" className="text-xs font-mono">
                        Byte&apos;s Hints
                      </TabsTrigger>
                      <TabsTrigger value="similar" className="text-xs font-mono">
                        Similar Problems
                      </TabsTrigger>
                    </TabsList>
                  </div>

                  <TabsContent value="description" className="mt-0 min-h-0 flex-1 overflow-y-auto p-4 sm:p-6">
                    {state.phase === "starting" && !state.problem ? (
                      <div className="space-y-3" aria-busy="true">
                        <Skeleton className="h-8 w-2/3" />
                        <Skeleton className="h-4 w-full" />
                        <Skeleton className="h-4 w-5/6" />
                        <Skeleton className="h-64 w-full" />
                      </div>
                    ) : (
                      <div className="space-y-6">
                        <ProblemStatementPanel problem={state.problem} />
                      </div>
                    )}
                  </TabsContent>

                  <TabsContent value="hints" className="mt-0 min-h-0 flex-1 overflow-y-auto p-4 sm:p-6">
                    <HintPanel hint={hint} />
                  </TabsContent>

                  <TabsContent value="similar" className="mt-0 min-h-0 flex-1 overflow-y-auto p-4 sm:p-6">
                    {state.problem ? (
                      <SimilarProblems problemId={state.problem.id} />
                    ) : (
                      <p className="font-mono text-xs text-tv-text-muted">No active problem.</p>
                    )}
                  </TabsContent>
                </Tabs>
              </div>

              {/* Right Column: Code Editor & Execution Console */}
              <div className="flex h-full min-h-0 flex-col bg-tv-surface-deep overflow-hidden">
                {state.phase === "exhausted" ? (
                  <div className="flex h-full items-center justify-center p-6">
                    <div className="glass-panel max-w-md rounded-tv-card p-6 text-center">
                      <p className="mb-4 text-sm text-tv-text-body">
                        You&apos;ve completed all available problems in this scope!
                      </p>
                      <Button onClick={() => void endSession()}>View Session Summary</Button>
                    </div>
                  </div>
                ) : (
                  <div className="flex h-full min-h-0 flex-col">
                    {/* Upper: Monaco Code Editor */}
                    <div className="min-h-0 flex-1 overflow-hidden">
                      <AnswerSurface
                        languages={LANGUAGES}
                        language={state.language}
                        onLanguageChange={setLanguage}
                        code={state.code}
                        onCodeChange={setCode}
                        problem={state.problem}
                        disabled={isBusy}
                        className="h-full rounded-none border-0 border-b border-tv-border shadow-none"
                      />
                    </div>

                    {/* Lower: Verdict / Submission Console */}
                    <div className="shrink-0 bg-tv-surface p-3 border-t border-tv-border">
                      {state.phase === "judge0" && state.judge0StartedAt !== null ? (
                        <div className="p-3">
                          <Judge0Wait startedAt={state.judge0StartedAt} />
                        </div>
                      ) : state.phase === "result" && state.lastResult ? (
                        <div className="space-y-3">
                          <VerdictPanel result={state.lastResult} />
                          <div className="flex items-center gap-3">
                            {state.lastResult.next_problem ? (
                              <Button onClick={advanceToNext} className="gap-2">
                                Next Problem &rarr;
                              </Button>
                            ) : (
                              <Button onClick={() => void endSession()} className="gap-2">
                                View Session Summary
                              </Button>
                            )}
                          </div>
                        </div>
                      ) : (
                        <div className="flex items-center justify-between gap-3">
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => void skip()}
                            disabled={isBusy}
                            className="font-mono text-xs text-tv-text-muted hover:text-tv-text-hi"
                          >
                            Skip Problem
                          </Button>
                          <Button
                            onClick={() => void submitAnswer()}
                            disabled={isBusy || !state.code.trim()}
                            size="sm"
                            className="font-mono text-xs"
                          >
                            Submit Solution
                          </Button>
                        </div>
                      )}
                    </div>
                  </div>
                )}
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
