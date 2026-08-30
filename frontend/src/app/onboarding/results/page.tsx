"use client";

/**
 * `/onboarding/results` — item 7 of plan.md's route map. Presents the
 * confirmed vs. debunked skill hypotheses from the quiz's close-out
 * (`CloseSessionResponse.per_topic_breakdown`) — the product's core USP, so
 * this is the moment that has to land — then feeds those two buckets into
 * `POST /roadmap/generate` and navigates to ATLAS's `/roadmap`.
 *
 * Reads its data from `readCachedCloseResult` (see
 * `components/practice/session-cache.ts`) rather than re-fetching: the quiz
 * page is the only place `POST /practice/close` is called, and
 * `GetSessionResponse` (the only other way to look up a session) doesn't
 * carry `per_topic_breakdown` at all. If the cache is empty — direct nav,
 * reload, a different tab — this degrades to an explicit empty state
 * pointing back at the quiz rather than guessing at numbers.
 */
import { Suspense, useMemo, useState, type ReactNode } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useMutation } from "@tanstack/react-query";
import { AlertTriangle } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { generateRoadmap } from "@/lib/api/endpoints";
import { ApiError } from "@/lib/api/client";
import { formatMasteryScore, formatPercent, topicLabel } from "@/components/practice/format";
import { readCachedCloseResult } from "@/components/practice/session-cache";
import { ThetaDelta } from "@/components/practice/theta-gauge";
import { TopicMasteryChart } from "@/components/practice/topic-mastery-chart";
import { QuizHeader } from "@/components/quiz/quiz-header";
import { HYPOTHESIS_CONFIRM_THRESHOLD } from "@/components/quiz/seed-topics";

const TARGET_ROLES = [
  "Software Engineer - DSA & Problem Solving",
  "Backend Engineer",
  "Competitive Programmer",
  "ML / Data Engineer",
];

export default function ResultsPage() {
  return (
    <Suspense fallback={null}>
      <ResultsPageInner />
    </Suspense>
  );
}

function ResultsPageInner() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const sessionId = searchParams.get("session");
  const [targetRole, setTargetRole] = useState(
    TARGET_ROLES[0] ?? "Software Engineer - DSA & Problem Solving",
  );

  const closeResult = useMemo(
    () => (sessionId ? readCachedCloseResult(sessionId) : null),
    [sessionId],
  );

  const { confirmed, debunked, allTopics } = useMemo(() => {
    if (!closeResult) return { confirmed: [] as string[], debunked: [] as string[], allTopics: [] as string[] };
    const topics = Object.values(closeResult.per_topic_breakdown);
    const confirmedTopics = topics
      .filter((t) => t.mastery_score >= HYPOTHESIS_CONFIRM_THRESHOLD)
      .map((t) => t.topic);
    const debunkedTopics = topics
      .filter((t) => t.mastery_score < HYPOTHESIS_CONFIRM_THRESHOLD)
      .map((t) => t.topic);
    return {
      confirmed: confirmedTopics,
      debunked: debunkedTopics,
      allTopics: topics.map((t) => t.topic),
    };
  }, [closeResult]);

  const generateMutation = useMutation({
    mutationFn: () =>
      generateRoadmap({
        target_role: targetRole,
        confirmed_hypotheses: confirmed,
        debunked_hypotheses: debunked,
      }),
    onSuccess: () => router.push("/roadmap"),
  });

  if (!sessionId || !closeResult) {
    return (
      <div className="min-h-full bg-tv-bg-page">
        <div className="mx-auto max-w-[1280px] px-6 py-10 md:px-12">
          <QuizHeader eyebrow="Onboarding · Step 3 of 3" title="Your Results" className="mb-6" />
          <div className="glass-panel rounded-tv-card p-8 text-center">
            <p className="mb-4 text-sm text-tv-text-body">
              We couldn&apos;t find results for this quiz session in this browser tab. Results are
              only available immediately after finishing the diagnostic quiz.
            </p>
            <Button render={<Link href="/onboarding/quiz" />}>Retake the quiz</Button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-full bg-tv-bg-page">
      <div className="mx-auto max-w-[1280px] px-6 py-10 md:px-12">
        <QuizHeader
          eyebrow="Onboarding · Step 3 of 3"
          title="Your Results"
          subtitle="Here's what the diagnostic actually verified — not what you told us, what you demonstrated."
          className="mb-8"
        />

        <div className="mb-8 grid grid-cols-2 gap-4 md:grid-cols-4">
          <StatTile label="θ start → final">
            <ThetaDelta before={closeResult.theta_start} after={closeResult.theta_final} />
          </StatTile>
          <StatTile label="Accuracy" value={formatPercent(closeResult.accuracy)} />
          <StatTile label="Mastery score" value={formatMasteryScore(closeResult.mastery_score)} />
          <StatTile
            label="Solved"
            value={`${closeResult.total_solved} / ${closeResult.total_questions}`}
          />
        </div>

        <div className="mb-8 grid grid-cols-1 gap-6 md:grid-cols-2">
          <Card className="glow-card-cyan">
            <CardHeader>
              <CardTitle className="font-display text-h4 uppercase text-tv-cyan">
                Confirmed strengths
              </CardTitle>
            </CardHeader>
            <CardContent>
              {confirmed.length === 0 ? (
                <p className="text-sm text-tv-text-body">
                  Nothing cleared the bar yet — that&apos;s exactly what the roadmap is for.
                </p>
              ) : (
                <div className="flex flex-wrap gap-2">
                  {confirmed.map((topic) => (
                    <Badge key={topic} variant="success" className="rounded-tv-pill">
                      ✓ {topicLabel(topic)} ·{" "}
                      {formatMasteryScore(closeResult.per_topic_breakdown[topic]?.mastery_score ?? 0)}
                    </Badge>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>

          <Card className="glow-card-rose">
            <CardHeader>
              <CardTitle className="font-display text-h4 uppercase text-tv-rose">
                Debunked gaps
              </CardTitle>
            </CardHeader>
            <CardContent>
              {debunked.length === 0 ? (
                <p className="text-sm text-tv-text-body">No gaps surfaced in this pass.</p>
              ) : (
                <div className="flex flex-wrap gap-2">
                  {debunked.map((topic) => (
                    <Badge key={topic} variant="error" className="rounded-tv-pill">
                      ✕ {topicLabel(topic)} ·{" "}
                      {formatMasteryScore(closeResult.per_topic_breakdown[topic]?.mastery_score ?? 0)}
                    </Badge>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        </div>

        <Card className="mb-8">
          <CardHeader>
            <CardTitle className="font-display text-h4 uppercase text-tv-text-hi">
              Measured mastery per topic
            </CardTitle>
          </CardHeader>
          <CardContent>
            <TopicMasteryChart
              data={allTopics.map((topic) => ({
                topic,
                masteryScore: closeResult.per_topic_breakdown[topic]?.mastery_score ?? 0,
              }))}
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="font-display text-h4 uppercase text-tv-text-hi">
              Build my roadmap
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex flex-col gap-1.5">
              <label htmlFor="target-role" className="font-mono text-xs text-tv-text-body uppercase">
                Target role
              </label>
              <Select
                value={targetRole}
                onValueChange={(value) => {
                  if (value) setTargetRole(value);
                }}
              >
                <SelectTrigger id="target-role" className="w-full sm:w-80">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {TARGET_ROLES.map((role) => (
                    <SelectItem key={role} value={role}>
                      {role}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            {generateMutation.isError && (
              <p className="flex items-center gap-2 rounded-tv-btn border border-tv-rose/30 bg-tv-rose/10 px-3 py-2 text-sm text-tv-rose">
                <AlertTriangle className="size-4 shrink-0" aria-hidden />
                {generateMutation.error instanceof ApiError
                  ? generateMutation.error.message
                  : "Couldn't generate your roadmap — try again."}
              </p>
            )}

            <Button
              onClick={() => generateMutation.mutate()}
              disabled={generateMutation.isPending}
            >
              {generateMutation.isPending ? "Generating…" : "Generate my roadmap"}
            </Button>
          </CardContent>
        </Card>
      </div>
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
