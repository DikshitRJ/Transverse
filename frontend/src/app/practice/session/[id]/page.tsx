"use client";

/**
 * `/practice/session/[id]` — session summary + Learning DNA (plan.md route
 * 15). Combines three sources:
 *
 *  - `GET /practice/session/{id}` — the response history (always fetched;
 *    source of truth for the theta-over-time chart)
 *  - the `CloseSessionResponse` cached by `use-practice-engine.ts`'s
 *    `endSession()` when THIS tab is the one that closed the session (see
 *    `session-cache.ts`) — the only source for `per_topic_breakdown` /
 *    `mastery_score` / `accuracy`, which no GET endpoint re-derives
 *  - `GET /user/profile` — the learner's overall Learning DNA, which is
 *    profile-level, not session-scoped, but belongs on this page per the brief
 *
 * When the cache misses (direct link, reload, a different tab), per-topic
 * mastery falls back to a client-computed estimate from the response list
 * (see `lookupTopicForProblem`) and is labeled as an estimate rather than
 * presented as the server's real number.
 */
import { useMemo, type ReactNode } from "react";
import { useParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Footer } from "@/components/shell/footer";
import { PageContainer } from "@/components/shell/page-container";
import { TopNav } from "@/components/shell/top-nav";
import { formatDuration, formatMasteryScore, formatPercent } from "@/components/practice/format";
import { readCachedCloseResult } from "@/components/practice/session-cache";
import { ThetaDelta } from "@/components/practice/theta-gauge";
import { ThetaHistoryChart } from "@/components/practice/theta-history-chart";
import { TopicMasteryChart } from "@/components/practice/topic-mastery-chart";
import { lookupTopicForProblem } from "@/components/practice/topic-lookup";
import { getErrorAnalysis, getPracticeSession, getUserProfile } from "@/lib/api/endpoints";

export default function PracticeSessionSummaryPage() {
  const params = useParams<{ id: string }>();
  const sessionId = params.id;

  const sessionQuery = useQuery({
    queryKey: ["practice", "session", sessionId],
    queryFn: () => getPracticeSession(sessionId),
    enabled: Boolean(sessionId),
  });

  const errorAnalysisQuery = useQuery({
    queryKey: ["practice", "error-analysis", sessionId],
    queryFn: () => getErrorAnalysis(sessionId),
    enabled: Boolean(sessionId),
  });

  const profileQuery = useQuery({
    queryKey: ["user", "profile"],
    queryFn: getUserProfile,
  });

  const cached = useMemo(() => (sessionId ? readCachedCloseResult(sessionId) : null), [sessionId]);
  const session = sessionQuery.data;

  const stats = useMemo(() => {
    if (cached) {
      return {
        thetaStart: cached.theta_start,
        thetaFinal: cached.theta_final,
        accuracy: cached.accuracy,
        masteryScore: cached.mastery_score,
        totalQuestions: cached.total_questions,
        totalSolved: cached.total_solved,
        perTopic: Object.values(cached.per_topic_breakdown).map((t) => ({
          topic: t.topic,
          masteryScore: t.mastery_score,
        })),
        isEstimated: false,
      };
    }
    if (!session) return null;
    const responses = session.responses;
    const total = responses.length;
    const solved = responses.filter((r) => r.is_correct).length;
    const accuracy = total > 0 ? solved / total : 0;
    const byTopic = new Map<string, { attempts: number; correct: number }>();
    for (const r of responses) {
      const topic = lookupTopicForProblem(r.problem_id);
      const entry = byTopic.get(topic) ?? { attempts: 0, correct: 0 };
      entry.attempts += 1;
      if (r.is_correct) entry.correct += 1;
      byTopic.set(topic, entry);
    }
    // `mastery_score` is a 0-100 scale everywhere in the API (verified
    // against `practice_analytics.go:10`), so this client-side estimate —
    // built from a 0-1 correct/attempts ratio, since there's no server
    // number to fall back on here — is scaled up to match, not left as a
    // fraction next to real 0-100 numbers.
    return {
      thetaStart: session.theta_start,
      thetaFinal: session.theta_current,
      accuracy,
      masteryScore: accuracy * 100,
      totalQuestions: total,
      totalSolved: solved,
      perTopic: Array.from(byTopic.entries()).map(([topic, e]) => ({
        topic,
        masteryScore: e.attempts > 0 ? (e.correct / e.attempts) * 100 : 0,
      })),
      isEstimated: true,
    };
  }, [cached, session]);

  const errorAnalysis = errorAnalysisQuery.data;
  const weakConcepts = Array.isArray(errorAnalysis?.weak_concepts)
    ? (errorAnalysis.weak_concepts as unknown[]).filter((c): c is string => typeof c === "string")
    : [];
  const recommendation =
    typeof errorAnalysis?.recommendation === "string" ? errorAnalysis.recommendation : undefined;

  const dna = profileQuery.data?.dna;

  return (
    <div className="flex min-h-full flex-col bg-tv-bg-page">
      <TopNav />
      <PageContainer className="flex-1">
        <h1 className="mb-8 font-display text-h1 font-bold tracking-[-1px] text-tv-text-hi uppercase">
          Session Summary
        </h1>

        {sessionQuery.isLoading && (
          <div className="space-y-4" aria-busy="true">
            <Skeleton className="h-24 w-full" />
            <Skeleton className="h-64 w-full" />
          </div>
        )}

        {sessionQuery.isError && (
          <div className="glass-panel rounded-tv-card p-8 text-center">
            <p className="text-sm text-tv-rose">
              Couldn&apos;t load this session — it may not exist, or you may not have access to it.
            </p>
          </div>
        )}

        {session && stats && (
          <div className="space-y-8">
            <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
              <StatTile label="θ start → final">
                <ThetaDelta before={stats.thetaStart} after={stats.thetaFinal} />
              </StatTile>
              <StatTile label="Accuracy" value={formatPercent(stats.accuracy)} />
              <StatTile
                label={stats.isEstimated ? "Mastery (est.)" : "Mastery score"}
                value={formatMasteryScore(stats.masteryScore)}
              />
              <StatTile
                label="Solved"
                value={`${stats.totalSolved} / ${stats.totalQuestions}`}
              />
            </div>

            <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
              <Card>
                <CardHeader>
                  <CardTitle className="font-display text-h4 uppercase text-tv-text-hi">
                    Theta over time
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <ThetaHistoryChart thetaStart={stats.thetaStart} responses={session.responses} />
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle className="font-display text-h4 uppercase text-tv-text-hi">
                    Per-topic mastery
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <TopicMasteryChart data={stats.perTopic} />
                  {stats.isEstimated && (
                    <p className="mt-2 font-mono text-[10px] text-tv-text-body">
                      Estimated from this session&apos;s responses — reopen this page right after
                      ending a session from `/practice` for the server-computed breakdown.
                    </p>
                  )}
                </CardContent>
              </Card>
            </div>

            <Card>
              <CardHeader>
                <CardTitle className="font-display text-h4 uppercase text-tv-text-hi">
                  Error analysis
                </CardTitle>
              </CardHeader>
              <CardContent>
                {errorAnalysisQuery.isLoading && (
                  <div className="space-y-2" aria-busy="true">
                    <Skeleton className="h-4 w-2/3" />
                    <Skeleton className="h-4 w-1/2" />
                  </div>
                )}
                {errorAnalysisQuery.isError && (
                  <p className="text-sm text-tv-rose">Couldn&apos;t load error analysis.</p>
                )}
                {errorAnalysis && errorAnalysis.status !== "done" && (
                  <p className="text-sm text-tv-text-body">
                    {errorAnalysis.message ??
                      "Error analysis results will appear here once enough misses accumulate."}
                  </p>
                )}
                {errorAnalysis && errorAnalysis.status === "done" && (
                  <div className="space-y-3">
                    {weakConcepts.length > 0 && (
                      <div className="flex flex-wrap gap-2">
                        {weakConcepts.map((c) => (
                          <Badge key={c} variant="warning">
                            {c}
                          </Badge>
                        ))}
                      </div>
                    )}
                    {recommendation && <p className="text-sm text-tv-text-body">{recommendation}</p>}
                  </div>
                )}
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle className="font-display text-h4 uppercase text-tv-text-hi">
                  Learning DNA
                </CardTitle>
              </CardHeader>
              <CardContent>
                {profileQuery.isLoading && (
                  <div className="grid grid-cols-2 gap-3 md:grid-cols-4" aria-busy="true">
                    {Array.from({ length: 8 }).map((_, i) => (
                      <Skeleton key={i} className="h-16 w-full" />
                    ))}
                  </div>
                )}
                {profileQuery.isError && (
                  <p className="text-sm text-tv-rose">Couldn&apos;t load your Learning DNA.</p>
                )}
                {dna && (
                  <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
                    <StatTile label="Avg accuracy" value={formatPercent(dna.avg_accuracy)} />
                    <StatTile label="Avg time / problem" value={formatDuration(dna.avg_time_taken_ms)} />
                    <StatTile label="Carelessness index" value={formatPercent(dna.carelessness_index)} />
                    <StatTile label="Peak hour" value={`${dna.peak_performance_hour}:00`} />
                    <StatTile label="Total sessions" value={String(dna.total_sessions)} />
                    <StatTile label="Total solved" value={String(dna.total_problems_solved)} />
                    <StatTile label="Streak record" value={String(dna.streak_record)} />
                    <StatTile label="Preferred platform" value={dna.preferred_platform || "—"} />
                  </div>
                )}
              </CardContent>
            </Card>
          </div>
        )}
      </PageContainer>
      <Footer />
    </div>
  );
}

function StatTile({
  label,
  value,
  children,
}: {
  label: string;
  value?: string;
  children?: ReactNode;
}) {
  return (
    <div className="glass-panel rounded-tv-card p-4">
      <p className="mb-1 font-mono text-xs text-tv-text-body uppercase tracking-wide">{label}</p>
      {value !== undefined ? (
        <p className="font-display text-h3 font-bold text-tv-text-hi">{value}</p>
      ) : (
        children
      )}
    </div>
  );
}
