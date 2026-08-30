"use client";

import Image from "next/image";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState, type ReactNode } from "react";
import { useQuery } from "@tanstack/react-query";
import { CodeIcon, LoaderCircleIcon, LogOutIcon, SparklesIcon, TrophyIcon } from "lucide-react";
import { TopNav } from "@/components/shell/top-nav";
import { Footer } from "@/components/shell/footer";
import { PageContainer } from "@/components/shell/page-container";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Separator } from "@/components/ui/separator";
import { useAuth } from "@/components/providers/auth-provider";
import {
  connectCodeforcesEvidence,
  connectGithubEvidence,
  connectLeetcodeEvidence,
  getUserProfile,
} from "@/lib/api";
import { AUTHED_NAV_LINKS, AppNavActions } from "@/components/dashboard/app-nav";
import { SectionHeading } from "@/components/dashboard/section-heading";
import { EmptyPanel, ErrorPanel } from "@/components/dashboard/async-panels";
import { useChartMotionPreference } from "@/components/dashboard/use-chart-motion-preference";
import { ConnectorCard } from "./_components/connector-card";
import { getConnectorRecords } from "./_lib/connector-store";

export default function SettingsPage() {
  const { user, isAuthenticated, isLoading: authLoading, logout } = useAuth();
  const router = useRouter();
  const [loggingOut, setLoggingOut] = useState(false);
  const [reduceMotion, setReduceMotion] = useChartMotionPreference();
  const [connectorRecords, setConnectorRecords] = useState<ReturnType<typeof getConnectorRecords>>({});

  useEffect(() => {
    setConnectorRecords(getConnectorRecords());
  }, []);

  const profileQuery = useQuery({
    queryKey: ["profile"],
    queryFn: getUserProfile,
    enabled: isAuthenticated === true,
  });

  async function handleLogout() {
    setLoggingOut(true);
    try {
      await logout();
      router.push("/");
    } catch {
      setLoggingOut(false);
    }
  }

  if (!authLoading && !isAuthenticated) {
    return (
      <Shell>
        <PageContainer className="flex flex-1 items-center justify-center">
          <EmptyPanel
            title="Sign in to view settings"
            description="Connected accounts, preferences, and session controls all live here once you're signed in."
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

  return (
    <Shell>
      <PageContainer className="flex flex-1 flex-col gap-8">
        <div>
          <p className="font-mono text-sm text-tv-cyan uppercase">Settings</p>
          <h1 className="font-display text-h1 font-bold text-tv-text-hi">Account settings</h1>
        </div>

        <Card>
          <CardHeader>
            <CardTitle className="text-h4">Account</CardTitle>
            <CardDescription>Your identity on Transverse</CardDescription>
          </CardHeader>
          <CardContent>
            {profileQuery.isPending ? (
              <Skeleton className="h-16 w-full" />
            ) : profileQuery.isError ? (
              <ErrorPanel error={profileQuery.error} onRetry={() => profileQuery.refetch()} />
            ) : (
              <dl className="grid grid-cols-1 gap-4 sm:grid-cols-3">
                <div>
                  <dt className="font-mono text-[11px] text-tv-text-body uppercase">Username</dt>
                  <dd className="font-body text-sm text-tv-text-hi">{profileQuery.data.username}</dd>
                </div>
                <div>
                  <dt className="font-mono text-[11px] text-tv-text-body uppercase">Email</dt>
                  <dd className="font-body text-sm text-tv-text-hi">{profileQuery.data.email}</dd>
                </div>
                <div>
                  <dt className="font-mono text-[11px] text-tv-text-body uppercase">Member since</dt>
                  <dd className="font-body text-sm text-tv-text-hi">
                    {new Date(profileQuery.data.created_at).toLocaleDateString(undefined, {
                      dateStyle: "long",
                    })}
                  </dd>
                </div>
              </dl>
            )}
          </CardContent>
        </Card>

        <section className="flex flex-col gap-3">
          <SectionHeading
            title="Connected accounts"
            description="Sync evidence from your past work so Transverse can calibrate faster. There's no live status endpoint yet — connections shown here reflect the last request you made from this browser."
          />
          <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
            <ConnectorCard
              kind="github"
              label="GitHub"
              description="Analyze your public repos for skill evidence."
              fieldLabel="GitHub username"
              placeholder="octocat"
              icon={<Image src="/figma/github-mark.png" alt="" width={20} height={20} />}
              initialRecord={connectorRecords.github}
              connect={connectGithubEvidence}
            />
            <ConnectorCard
              kind="leetcode"
              label="LeetCode"
              description="Import your solved-problem history."
              fieldLabel="LeetCode username"
              placeholder="byte_learner"
              icon={<CodeIcon className="size-4" aria-hidden="true" />}
              initialRecord={connectorRecords.leetcode}
              connect={connectLeetcodeEvidence}
            />
            <ConnectorCard
              kind="codeforces"
              label="Codeforces"
              description="Import your contest & submission history."
              fieldLabel="Codeforces handle"
              placeholder="tourist"
              icon={<TrophyIcon className="size-4" aria-hidden="true" />}
              initialRecord={connectorRecords.codeforces}
              connect={connectCodeforcesEvidence}
            />
          </div>
        </section>

        <Card>
          <CardHeader>
            <CardTitle className="text-h4">Preferences</CardTitle>
            <CardDescription>Stored on this device only</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex items-center justify-between gap-4">
              <div className="flex items-start gap-3">
                <SparklesIcon className="mt-0.5 size-4 shrink-0 text-tv-cyan" aria-hidden="true" />
                <div>
                  <p className="font-body text-sm font-medium text-tv-text-hi">Reduce motion in charts</p>
                  <p className="font-body text-xs text-tv-text-body">
                    Turns off chart entrance animations on the dashboard and profile. This is in addition to your
                    OS-level reduced-motion setting, which Transverse always respects.
                  </p>
                </div>
              </div>
              <Button
                variant={reduceMotion ? "default" : "outline"}
                size="sm"
                onClick={() => setReduceMotion(!reduceMotion)}
                aria-pressed={reduceMotion}
              >
                {reduceMotion ? "On" : "Off"}
              </Button>
            </div>
          </CardContent>
        </Card>

        <Card className="border-tv-rose/30">
          <CardHeader>
            <CardTitle className="text-h4">Session</CardTitle>
            <CardDescription>Signed in as {user?.username ?? "—"}</CardDescription>
          </CardHeader>
          <CardContent>
            <Separator className="mb-4" />
            <Button variant="destructive" onClick={handleLogout} disabled={loggingOut}>
              {loggingOut ? <LoaderCircleIcon className="animate-spin" /> : <LogOutIcon />}
              Log out
            </Button>
          </CardContent>
        </Card>
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
