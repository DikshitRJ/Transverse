"use client";

import Link from "next/link";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { ArrowLeft, Clock, ExternalLink, Gauge } from "lucide-react";
import { ApiError, completeRoadmapNode, getRoadmap } from "@/lib/api";
import type { RoadmapCurrentResponse, Tutorial } from "@/lib/api/types";
import { PageContainer } from "@/components/shell/page-container";
import { TopNav } from "@/components/shell/top-nav";
import { Footer } from "@/components/shell/footer";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { SafeHtml } from "./safe-html";
import type { ReactNode } from "react";

export interface TutorialReaderProps {
  tutorialId: string;
}

function findTutorial(
  data: RoadmapCurrentResponse | undefined,
  tutorialId: string,
): { tutorial: Tutorial; nodeId: string; nodeTitle: string } | undefined {
  const subsections = data?.current_section?.subsections ?? [];
  for (const sub of subsections) {
    const tutorial = sub.tutorials.find((t) => t.id === tutorialId);
    if (tutorial) return { tutorial, nodeId: sub.node_id, nodeTitle: sub.title };
  }
  return undefined;
}

export function TutorialReader({ tutorialId }: TutorialReaderProps) {
  const queryClient = useQueryClient();

  // Tutorials aren't independently fetchable (plan.md §2, route 11: "from
  // roadmap payload") — only the active section's `GET /roadmap` response
  // ever carries full tutorial objects, so this reader locates its content
  // inside that same query rather than hitting a dedicated endpoint.
  const { data, isLoading, isError, error, refetch } = useQuery<RoadmapCurrentResponse, ApiError>({
    queryKey: ["roadmap"],
    queryFn: () => getRoadmap(),
  });

  const found = findTutorial(data, tutorialId);

  // No dedicated "mark tutorial read" endpoint exists — per the route
  // table, this reader's completion action *is* the node-completion call.
  const completeMutation = useMutation({
    mutationFn: (nodeId: string) => completeRoadmapNode(nodeId),
    onSuccess: async () => {
      toast.success("Marked complete — checking what's unlocked next.");
      await queryClient.invalidateQueries({ queryKey: ["roadmap"] });
    },
    onError: () => toast.error("Couldn't mark this complete. Try again."),
  });

  if (isLoading) {
    return (
      <Shell>
        <Skeleton className="h-8 w-32" />
        <Skeleton className="h-10 w-2/3" />
        <Skeleton className="h-48 w-full rounded-tv-card" />
      </Shell>
    );
  }

  if (isError) {
    return (
      <Shell>
        <ErrorState
          message={error instanceof ApiError ? error.message : "Something went wrong loading this tutorial."}
          onRetry={() => void refetch()}
        />
      </Shell>
    );
  }

  if (!found) {
    return (
      <Shell>
        <NotFoundState />
      </Shell>
    );
  }

  const { tutorial, nodeId, nodeTitle } = found;

  return (
    <Shell>
      <Link
        href={`/roadmap/node/${nodeId}`}
        className="flex w-fit items-center gap-1.5 font-mono text-xs text-tv-text-body hover:text-tv-cyan"
      >
        <ArrowLeft className="size-3.5" aria-hidden />
        Back to {nodeTitle}
      </Link>

      <article className="flex flex-col gap-6 rounded-tv-card border border-tv-border bg-tv-surface p-6">
        <header className="flex flex-col gap-3">
          <div className="flex flex-wrap items-center gap-2">
            <Badge variant="outline" className="capitalize">
              {tutorial.type}
            </Badge>
            <Badge variant="outline" className="capitalize">
              {tutorial.difficulty}
            </Badge>
            <Badge variant={tutorial.status === "COMPLETED" ? "success" : "locked"}>{tutorial.status}</Badge>
          </div>
          <h1 className="font-display text-h1 font-bold tracking-tight text-tv-text-hi uppercase">
            {tutorial.title}
          </h1>
          <div className="flex flex-wrap items-center gap-4 font-mono text-xs text-tv-text-body">
            <span className="flex items-center gap-1">
              <Clock className="size-3.5" aria-hidden />
              {tutorial.estimated_minutes} min read
            </span>
            <span className="flex items-center gap-1">
              <Gauge className="size-3.5" aria-hidden />
              Source: {tutorial.source}
            </span>
          </div>
        </header>

        {tutorial.thumbnail_url && (
          // Externally hosted thumbnail — plain <img>, not next/image (would
          // require remote-pattern config in next.config.ts, off-limits here).
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={tutorial.thumbnail_url}
            alt=""
            className="max-h-64 w-full rounded-tv-btn border border-tv-border object-cover"
          />
        )}

        <SafeHtml html={tutorial.summary} />

        {tutorial.license_note && (
          <p className="font-mono text-xs text-tv-text-body/70">{tutorial.license_note}</p>
        )}

        <div className="flex flex-wrap items-center gap-3 border-t border-tv-border pt-5">
          <Button render={<a href={tutorial.source_url} target="_blank" rel="noopener noreferrer" />}>
            <ExternalLink className="size-4" aria-hidden />
            Read on {tutorial.source}
          </Button>
          <Button
            variant="outline-cyan"
            onClick={() => completeMutation.mutate(nodeId)}
            disabled={completeMutation.isPending}
          >
            {completeMutation.isPending ? "Marking complete…" : "Mark Complete"}
          </Button>
        </div>
      </article>
    </Shell>
  );
}

function NotFoundState() {
  return (
    <div className="flex flex-col items-center gap-4 rounded-tv-card border border-tv-border bg-tv-surface px-8 py-16 text-center">
      <h1 className="font-display text-h2 font-bold text-tv-text-hi uppercase">Tutorial not found</h1>
      <p className="max-w-md text-sm text-tv-text-body">
        This tutorial isn&apos;t part of your current active section, or your roadmap has moved on.
      </p>
      <Button variant="outline" render={<Link href="/roadmap" />}>
        Back to roadmap
      </Button>
    </div>
  );
}

function ErrorState({ message, onRetry }: { message: string; onRetry: () => void }) {
  return (
    <div className="flex flex-col items-center gap-4 rounded-tv-card border border-tv-rose/30 bg-tv-surface px-8 py-16 text-center">
      <h1 className="font-display text-h2 font-bold text-tv-text-hi uppercase">Couldn&apos;t load this tutorial</h1>
      <p className="max-w-md text-sm text-tv-text-body">{message}</p>
      <Button variant="outline" onClick={onRetry}>
        Try again
      </Button>
    </div>
  );
}

function Shell({ children }: { children: ReactNode }) {
  return (
    <div className="flex min-h-full flex-col bg-tv-bg-page">
      <TopNav />
      <PageContainer className="flex max-w-[860px] flex-1 flex-col gap-6">{children}</PageContainer>
      <Footer />
    </div>
  );
}
