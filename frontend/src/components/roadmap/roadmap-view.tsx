"use client";

import { useEffect, useRef, useState, type ReactNode } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { motion, useReducedMotion } from "motion/react";
import { toast } from "sonner";
import { Compass, RefreshCw, Sparkles } from "lucide-react";
import { ApiError, generateRoadmap, getRoadmap } from "@/lib/api";
import type { NodeUnlockedEventData, RoadmapCurrentResponse } from "@/lib/api/types";
import { useTransverseEvent } from "@/components/providers/sse-provider";
import { PageContainer } from "@/components/shell/page-container";
import { TopNav } from "@/components/shell/top-nav";
import { Footer } from "@/components/shell/footer";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";
import { NodeCard } from "./node-card";
import { LockedSectionPreview } from "./locked-section-preview";
import { diffRoadmap, NO_TRANSITION, type RoadmapTransition } from "./roadmap-status";

const UNLOCK_ANIMATION_MS = 450;

export function RoadmapView() {
  const queryClient = useQueryClient();
  const reduceMotion = useReducedMotion();

  const { data, isLoading, isError, error, refetch, isFetching } = useQuery<RoadmapCurrentResponse, ApiError>({
    queryKey: ["roadmap"],
    queryFn: () => getRoadmap(),
  });

  const generateMutation = useMutation({
    mutationFn: () => generateRoadmap(),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["roadmap"] });
      toast.success("Your roadmap is ready.");
    },
    onError: () => toast.error("Couldn't generate a roadmap. Try again."),
  });

  // Drives the unlock choreography — see `diffRoadmap`. Fires off a snapshot
  // diff every time fresh data lands, regardless of whether the refetch was
  // triggered by SSE (`node.unlocked` / `roadmap.updated`, auto-invalidated
  // by SSEProvider) or by returning from a complete/test-out mutation on the
  // node detail page.
  const previousRef = useRef<RoadmapCurrentResponse | null>(null);
  const [transition, setTransition] = useState<RoadmapTransition>(NO_TRANSITION);

  useEffect(() => {
    if (!data) return;
    const next = diffRoadmap(previousRef.current, data);
    previousRef.current = data;
    if (next === NO_TRANSITION) return;
    setTransition(next);
    const timer = setTimeout(() => setTransition(NO_TRANSITION), reduceMotion ? 0 : UNLOCK_ANIMATION_MS);
    return () => clearTimeout(timer);
  }, [data, reduceMotion]);

  // The signature moment (plan.md §1.4): react to the real event, not just
  // the cache invalidation it causes — used for the toast; the visual
  // dissolve itself is driven by the diff above once the refetched data lands.
  useTransverseEvent<NodeUnlockedEventData>("node.unlocked", (event) => {
    toast(`${event.data.title} unlocked`, { icon: <Sparkles className="size-4 text-tv-cyan" /> });
  });

  if (isLoading) return <RoadmapShell><RoadmapSkeleton /></RoadmapShell>;

  if (isError) {
    if (error instanceof ApiError && error.status === 404) {
      return (
        <RoadmapShell>
          <RoadmapEmptyState onGenerate={() => generateMutation.mutate()} pending={generateMutation.isPending} />
        </RoadmapShell>
      );
    }
    return (
      <RoadmapShell>
        <RoadmapErrorState
          message={error instanceof ApiError ? error.message : "Something went wrong loading your roadmap."}
          onRetry={() => void refetch()}
        />
      </RoadmapShell>
    );
  }

  if (!data || !data.current_section) {
    return (
      <RoadmapShell>
        <RoadmapEmptyState onGenerate={() => generateMutation.mutate()} pending={generateMutation.isPending} />
      </RoadmapShell>
    );
  }

  const section = data.current_section;

  return (
    <RoadmapShell>
      <div className="flex flex-col gap-8">
        <motion.header
          className="flex flex-col gap-4 rounded-tv-card border border-tv-border-cyan/40 bg-tv-surface p-6"
          animate={
            transition.sectionChanged && !reduceMotion
              ? { boxShadow: ["0 0 0 rgba(0,242,255,0)", "0 0 30px rgba(0,242,255,0.35)", "0 0 15px rgba(0,242,255,0.1)"] }
              : undefined
          }
          transition={{ duration: 0.4, ease: "easeOut" }}
        >
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div className="flex flex-col gap-1">
              <span className="font-mono text-xs text-tv-cyan uppercase">
                Section {section.sequence} of {data.total_sections}
              </span>
              <h1 className="glow-text-cyan font-display text-h1 font-bold tracking-tight text-tv-text-hi uppercase">
                {section.title}
              </h1>
            </div>
            <Badge variant="success" className="rounded-tv-pill px-3 py-1 text-xs">
              {section.status}
            </Badge>
          </div>
          <div className="flex items-center gap-3">
            <div className="h-2 flex-1 overflow-hidden rounded-tv-pill bg-tv-surface-2">
              <div
                className="h-full rounded-tv-pill bg-tv-cyan transition-[width] duration-500 ease-out"
                style={{ width: `${Math.max(0, Math.min(100, Math.round(section.progress_percentage)))}%` }}
              />
            </div>
            <span className="font-mono text-xs text-tv-text-body">
              {Math.round(section.progress_percentage)}%
            </span>
          </div>
        </motion.header>

        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
          {section.subsections.map((sub) => (
            <NodeCard
              key={sub.node_id}
              subsection={sub}
              justMastered={transition.masteredNodeId === sub.node_id}
              justUnlocked={transition.unlockedNodeId === sub.node_id}
            />
          ))}
        </div>

        {data.upcoming_sections.length > 0 && (
          <div className="flex flex-col gap-3">
            <h2 className="font-mono text-xs tracking-wide text-tv-text-body uppercase">Up next</h2>
            <div className="flex flex-col gap-3">
              {data.upcoming_sections.map((up) => (
                <LockedSectionPreview key={up.sequence} section={up} />
              ))}
            </div>
          </div>
        )}

        {isFetching && (
          <div className="flex items-center gap-2 self-start font-mono text-xs text-tv-text-body">
            <RefreshCw className="size-3 animate-spin" aria-hidden />
            Syncing…
          </div>
        )}
      </div>
    </RoadmapShell>
  );
}

function RoadmapShell({ children }: { children: ReactNode }) {
  return (
    <div className="flex min-h-full flex-col bg-tv-bg-page">
      <TopNav />
      <PageContainer className="flex-1">{children}</PageContainer>
      <Footer />
    </div>
  );
}

function RoadmapSkeleton() {
  return (
    <div className="flex flex-col gap-8">
      <Skeleton className="h-32 w-full rounded-tv-card" />
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
        {Array.from({ length: 6 }).map((_, i) => (
          <Skeleton key={i} className="h-44 w-full rounded-tv-card" />
        ))}
      </div>
    </div>
  );
}

function RoadmapEmptyState({ onGenerate, pending }: { onGenerate: () => void; pending: boolean }) {
  return (
    <div className="flex flex-col items-center gap-4 rounded-tv-card border border-tv-border bg-tv-surface px-8 py-16 text-center">
      <Compass className="size-10 text-tv-cyan" aria-hidden />
      <h1 className="font-display text-h2 font-bold text-tv-text-hi uppercase">No roadmap yet</h1>
      <p className="max-w-md text-sm text-tv-text-body">
        We haven&apos;t built your learning path yet. Generate a roadmap to get a section-by-section plan,
        gated so you only see what you&apos;re ready for.
      </p>
      <Button onClick={onGenerate} disabled={pending}>
        {pending ? "Generating…" : "Generate my roadmap"}
      </Button>
    </div>
  );
}

function RoadmapErrorState({ message, onRetry }: { message: string; onRetry: () => void }) {
  return (
    <div className="flex flex-col items-center gap-4 rounded-tv-card border border-tv-rose/30 bg-tv-surface px-8 py-16 text-center">
      <h1 className="font-display text-h2 font-bold text-tv-text-hi uppercase">Couldn&apos;t load your roadmap</h1>
      <p className="max-w-md text-sm text-tv-text-body">{message}</p>
      <Button variant="outline" onClick={onRetry}>
        Try again
      </Button>
    </div>
  );
}
