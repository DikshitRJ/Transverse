"use client";

import Link from "next/link";
import type { ReactNode } from "react";
import { useQuery } from "@tanstack/react-query";
import { FlameIcon, GaugeIcon, TargetIcon, TrophyIcon } from "lucide-react";
import { TopNav } from "@/components/shell/top-nav";
import { Footer } from "@/components/shell/footer";
import { PageContainer } from "@/components/shell/page-container";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { useAuth } from "@/components/providers/auth-provider";
import { getPracticeTopics, getRoadmap, getUserHistory, getUserProfile } from "@/lib/api";
import { AUTHED_NAV_LINKS, AppNavActions } from "@/components/dashboard/app-nav";
import { SectionHeading } from "@/components/dashboard/section-heading";
import { EmptyPanel, ErrorPanel } from "@/components/dashboard/async-panels";
import { PrimaryActionBanner } from "@/components/dashboard/primary-action-banner";
import { RoadmapProgressCard } from "@/components/dashboard/roadmap-progress-card";
import { RecentSessionsList } from "@/components/dashboard/recent-sessions-list";
import { StatTile } from "@/components/charts/stat-tile";
import { TopicMasteryBarChart } from "@/components/charts/topic-mastery-bar-chart";

const HISTORY_LIMIT = 20;
const RECENT_SESSIONS_SHOWN = 5;
const TOP_TOPICS_SHOWN = 5;

export default function DashboardPage() {
  const { user, isAuthenticated, isLoading: authLoading } = useAuth();
  const queriesEnabled = isAuthenticated === true;

  const profileQuery = useQuery({
    queryKey: ["profile"],
    queryFn: getUserProfile,
    enabled: queriesEnabled,
  });
  const roadmapQuery = useQuery({
    queryKey: ["roadmap"],
    queryFn: getRoadmap,
    enabled: queriesEnabled,
  });
  const topicsQuery = useQuery({
    queryKey: ["practice", "topics"],
    queryFn: getPracticeTopics,
    enabled: queriesEnabled,
  });
  const historyQuery = useQuery({
    queryKey: ["history", { limit: HISTORY_LIMIT, offset: 0 }],
    queryFn: () => getUserHistory({ limit: HISTORY_LIMIT, offset: 0 }),
    enabled: queriesEnabled,
  });

  if (!authLoading && !isAuthenticated) {
    return (
      <Shell>
        <PageContainer className="flex flex-1 items-center justify-center">
          <EmptyPanel
            title="Sign in to view your dashboard"
            description="Your rating, roadmap progress, and practice history all live here once you're signed in."
            action={
              <Button render={<Link href="/signin" />} size="sm">
                Sign in
              </Button>
            }
          />
        </PageContainer>
      </Shell>
    );
  }

  const sessionsDesc = [...(historyQuery.data ?? [])].sort(
    (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime(),
  );

  const topTopics = [...(topicsQuery.data?.topics ?? [])]
    .filter((t) => t.attempt_count > 0)
    .sort((a, b) => b.mastery_score - a.mastery_score)
    .slice(0, TOP_TOPICS_SHOWN)
    // `mastery_score` is 0–100 on the wire (CalculateMasteryScore, practice_analytics.go)
    // — chart components expect a 0–1 fraction, matching formatPercent's contract.
    .map((t) => ({ topic: t.topic, masteryScore: t.mastery_score / 100, attemptCount: t.attempt_count }));

  return (
    <Shell>
      <PageContainer className="flex flex-1 flex-col gap-8">
        <div>
          <p className="font-mono text-sm text-tv-cyan uppercase">Dashboard</p>
          <h1 className="font-display text-h1 font-bold text-tv-text-hi">
            {authLoading || !user ? "Welcome back" : `Welcome back, ${user.username}`}
          </h1>
        </div>

        {roadmapQuery.isError ? (
          <ErrorPanel error={roadmapQuery.error} onRetry={() => roadmapQuery.refetch()} />
        ) : (
          <PrimaryActionBanner roadmap={roadmapQuery.data} />
        )}

        <section aria-label="Key stats" className="grid grid-cols-2 gap-4 lg:grid-cols-4">
          {profileQuery.isPending ? (
            <StatTilesSkeleton />
          ) : profileQuery.isError ? (
            <div className="col-span-full">
              <ErrorPanel error={profileQuery.error} onRetry={() => profileQuery.refetch()} />
            </div>
          ) : (
            <>
              <StatTile
                icon={<TrophyIcon className="size-4" aria-hidden="true" />}
                label="Glicko rating"
                value={Math.round(profileQuery.data.glicko_rating).toLocaleString()}
                hint={`± ${Math.round(profileQuery.data.glicko_rd)} RD`}
              />
              <StatTile
                icon={<GaugeIcon className="size-4" aria-hidden="true" />}
                label="Ability (θ)"
                value={profileQuery.data.theta.toFixed(2)}
              />
              <StatTile
                icon={<TargetIcon className="size-4" aria-hidden="true" />}
                label="Problems solved"
                value={profileQuery.data.dna.total_problems_solved.toLocaleString()}
              />
              <StatTile
                icon={<FlameIcon className="size-4" aria-hidden="true" />}
                label="Best streak"
                value={String(profileQuery.data.dna.streak_record)}
                hint="sessions"
              />
            </>
          )}
        </section>

        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle className="text-h4">Roadmap progress</CardTitle>
            </CardHeader>
            <CardContent>
              {roadmapQuery.isPending ? (
                <div className="flex flex-col gap-3">
                  <Skeleton className="h-4 w-1/2" />
                  <Skeleton className="h-2 w-full" />
                  <Skeleton className="h-16 w-full" />
                </div>
              ) : roadmapQuery.isError ? (
                <ErrorPanel error={roadmapQuery.error} onRetry={() => roadmapQuery.refetch()} />
              ) : (
                <RoadmapProgressCard roadmap={roadmapQuery.data} />
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-h4">Topic mastery</CardTitle>
            </CardHeader>
            <CardContent>
              {topicsQuery.isPending ? (
                <Skeleton className="h-52 w-full" />
              ) : topicsQuery.isError ? (
                <ErrorPanel error={topicsQuery.error} onRetry={() => topicsQuery.refetch()} />
              ) : (
                <TopicMasteryBarChart data={topTopics} />
              )}
            </CardContent>
          </Card>
        </div>

        <section className="flex flex-col gap-3">
          <SectionHeading
            title="Recent sessions"
            description="Your last few practice runs"
            action={
              <Button render={<Link href="/profile" />} variant="ghost" size="sm">
                View all history
              </Button>
            }
          />
          {historyQuery.isPending ? (
            <div className="flex flex-col gap-2">
              <Skeleton className="h-14 w-full" />
              <Skeleton className="h-14 w-full" />
              <Skeleton className="h-14 w-full" />
            </div>
          ) : historyQuery.isError ? (
            <ErrorPanel error={historyQuery.error} onRetry={() => historyQuery.refetch()} />
          ) : (
            <RecentSessionsList
              sessions={sessionsDesc.slice(0, RECENT_SESSIONS_SHOWN)}
              emptyAction={
                <Button render={<Link href="/practice" />} size="sm">
                  Start practicing
                </Button>
              }
            />
          )}
        </section>
      </PageContainer>
    </Shell>
  );
}

function Shell({ children }: { children: ReactNode }) {
  return (
    <div className="flex min-h-full flex-col bg-tv-bg">
      <TopNav links={AUTHED_NAV_LINKS} actions={<AppNavActions />} />
      {children}
      <Footer />
    </div>
  );
}

function StatTilesSkeleton() {
  return (
    <>
      {Array.from({ length: 4 }).map((_, i) => (
        <Skeleton key={i} className="h-[104px] w-full rounded-tv-card" />
      ))}
    </>
  );
}
