"use client";

import Link from "next/link";
import { useState, type ReactNode } from "react";
import { useQuery } from "@tanstack/react-query";
import { ChevronLeftIcon, ChevronRightIcon } from "lucide-react";
import { TopNav } from "@/components/shell/top-nav";
import { Footer } from "@/components/shell/footer";
import { PageContainer } from "@/components/shell/page-container";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { useAuth } from "@/components/providers/auth-provider";
import { getPracticeTopics, getUserHistory, getUserProfile } from "@/lib/api";
import { AUTHED_NAV_LINKS, AppNavActions } from "@/components/dashboard/app-nav";
import { SectionHeading } from "@/components/dashboard/section-heading";
import { EmptyPanel, ErrorPanel } from "@/components/dashboard/async-panels";
import { RecentSessionsList } from "@/components/dashboard/recent-sessions-list";
import { summarizeSession } from "@/components/dashboard/session-utils";
import { RatingTrendChart } from "@/components/charts/rating-trend-chart";
import { AccuracyVolumeChart } from "@/components/charts/accuracy-volume-chart";
import { TopicMasteryRadarChart } from "@/components/charts/topic-mastery-radar-chart";
import { TopicMasteryBarChart } from "@/components/charts/topic-mastery-bar-chart";
import { LearningDnaPanel } from "./_components/learning-dna-panel";

const TREND_LIMIT = 50;
const PAGE_SIZE = 10;
const RADAR_TOPIC_CAP = 8;

export default function ProfilePage() {
  const { isAuthenticated, isLoading: authLoading } = useAuth();
  const queriesEnabled = isAuthenticated === true;
  const [offset, setOffset] = useState(0);

  const profileQuery = useQuery({
    queryKey: ["profile"],
    queryFn: getUserProfile,
    enabled: queriesEnabled,
  });
  const topicsQuery = useQuery({
    queryKey: ["practice", "topics"],
    queryFn: getPracticeTopics,
    enabled: queriesEnabled,
  });
  const trendHistoryQuery = useQuery({
    queryKey: ["history", { limit: TREND_LIMIT, offset: 0 }],
    queryFn: () => getUserHistory({ limit: TREND_LIMIT, offset: 0 }),
    enabled: queriesEnabled,
  });
  const pageHistoryQuery = useQuery({
    queryKey: ["history", { limit: PAGE_SIZE, offset }],
    queryFn: () => getUserHistory({ limit: PAGE_SIZE, offset }),
    enabled: queriesEnabled,
    placeholderData: (prev) => prev,
  });

  if (!authLoading && !isAuthenticated) {
    return (
      <Shell>
        <PageContainer className="flex flex-1 items-center justify-center">
          <EmptyPanel
            title="Sign in to view your profile"
            description="Rating history, topic mastery, and your Learning DNA all live here once you're signed in."
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

  const trendSessionsAsc = [...(trendHistoryQuery.data ?? [])].sort(
    (a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime(),
  );
  const ratingTrendData = trendSessionsAsc.map((s) => ({
    date: s.created_at,
    theta: s.theta_current,
    questionCount: s.question_count,
  }));
  const accuracyVolumeData = trendSessionsAsc.map((s) => {
    const summary = summarizeSession(s);
    return { date: s.created_at, accuracy: summary.accuracy, volume: s.question_count };
  });

  const attemptedTopics = [...(topicsQuery.data?.topics ?? [])]
    .filter((t) => t.attempt_count > 0)
    .sort((a, b) => b.mastery_score - a.mastery_score)
    // `mastery_score` is 0–100 on the wire (CalculateMasteryScore, practice_analytics.go)
    // — chart components expect a 0–1 fraction, matching formatPercent's contract.
    .map((t) => ({ topic: t.topic, masteryScore: t.mastery_score / 100, attemptCount: t.attempt_count }));

  const canGoNext = (pageHistoryQuery.data?.length ?? 0) === PAGE_SIZE;

  return (
    <Shell>
      <PageContainer className="flex flex-1 flex-col gap-8">
        <div>
          <p className="font-mono text-sm text-tv-cyan uppercase">Profile</p>
          {profileQuery.isPending ? (
            <Skeleton className="h-9 w-64" />
          ) : profileQuery.isError ? (
            <h1 className="font-display text-h1 font-bold text-tv-text-hi">Profile</h1>
          ) : (
            <>
              <h1 className="font-display text-h1 font-bold text-tv-text-hi">{profileQuery.data.username}</h1>
              <p className="font-body text-sm text-tv-text-body">
                {profileQuery.data.email} · Member since{" "}
                {new Date(profileQuery.data.created_at).toLocaleDateString(undefined, {
                  month: "long",
                  year: "numeric",
                })}
              </p>
            </>
          )}
        </div>

        <Card>
          <CardHeader>
            <CardTitle className="text-h4">Ability progression</CardTitle>
          </CardHeader>
          <CardContent>
            {trendHistoryQuery.isPending ? (
              <Skeleton className="h-60 w-full" />
            ) : trendHistoryQuery.isError ? (
              <ErrorPanel error={trendHistoryQuery.error} onRetry={() => trendHistoryQuery.refetch()} />
            ) : (
              <RatingTrendChart data={ratingTrendData} />
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-h4">Accuracy & volume over time</CardTitle>
          </CardHeader>
          <CardContent>
            {trendHistoryQuery.isPending ? (
              <Skeleton className="h-44 w-full" />
            ) : trendHistoryQuery.isError ? (
              <ErrorPanel error={trendHistoryQuery.error} onRetry={() => trendHistoryQuery.refetch()} />
            ) : (
              <AccuracyVolumeChart data={accuracyVolumeData} />
            )}
          </CardContent>
        </Card>

        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle className="text-h4">Skill profile</CardTitle>
            </CardHeader>
            <CardContent>
              {topicsQuery.isPending ? (
                <Skeleton className="h-72 w-full" />
              ) : topicsQuery.isError ? (
                <ErrorPanel error={topicsQuery.error} onRetry={() => topicsQuery.refetch()} />
              ) : (
                <TopicMasteryRadarChart data={attemptedTopics.slice(0, RADAR_TOPIC_CAP)} />
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-h4">Mastery by topic</CardTitle>
            </CardHeader>
            <CardContent>
              {topicsQuery.isPending ? (
                <Skeleton className="h-72 w-full" />
              ) : topicsQuery.isError ? (
                <ErrorPanel error={topicsQuery.error} onRetry={() => topicsQuery.refetch()} />
              ) : (
                <TopicMasteryBarChart data={attemptedTopics} />
              )}
            </CardContent>
          </Card>
        </div>

        <Card>
          <CardHeader>
            <CardTitle className="text-h4">Learning DNA</CardTitle>
          </CardHeader>
          <CardContent>
            {profileQuery.isPending ? (
              <Skeleton className="h-40 w-full" />
            ) : profileQuery.isError ? (
              <ErrorPanel error={profileQuery.error} onRetry={() => profileQuery.refetch()} />
            ) : (
              <LearningDnaPanel dna={profileQuery.data.dna} />
            )}
          </CardContent>
        </Card>

        <section className="flex flex-col gap-3">
          <SectionHeading title="Session history" description="Every practice session, most recent first" />
          {pageHistoryQuery.isPending ? (
            <div className="flex flex-col gap-2">
              <Skeleton className="h-14 w-full" />
              <Skeleton className="h-14 w-full" />
              <Skeleton className="h-14 w-full" />
            </div>
          ) : pageHistoryQuery.isError ? (
            <ErrorPanel error={pageHistoryQuery.error} onRetry={() => pageHistoryQuery.refetch()} />
          ) : (
            <>
              <RecentSessionsList
                sessions={pageHistoryQuery.data ?? []}
                emptyAction={
                  offset === 0 ? (
                    <Button render={<Link href="/practice" />} size="sm">
                      Start practicing
                    </Button>
                  ) : undefined
                }
              />
              {(pageHistoryQuery.data?.length ?? 0) > 0 && (
                <div className="flex items-center justify-between pt-2">
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={offset === 0}
                    onClick={() => setOffset((o) => Math.max(0, o - PAGE_SIZE))}
                  >
                    <ChevronLeftIcon />
                    Previous
                  </Button>
                  <span className="font-mono text-xs text-tv-text-body">
                    Showing {offset + 1}–{offset + (pageHistoryQuery.data?.length ?? 0)}
                  </span>
                  <Button variant="outline" size="sm" disabled={!canGoNext} onClick={() => setOffset((o) => o + PAGE_SIZE)}>
                    Next
                    <ChevronRightIcon />
                  </Button>
                </div>
              )}
            </>
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
