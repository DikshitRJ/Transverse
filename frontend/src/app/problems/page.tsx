"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { ChevronLeftIcon, ChevronRightIcon, LinkIcon, SearchIcon, XIcon } from "lucide-react";
import { TopNav } from "@/components/shell/top-nav";
import { Footer } from "@/components/shell/footer";
import { PageContainer } from "@/components/shell/page-container";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useAuth } from "@/components/providers/auth-provider";
import { searchProblems } from "@/lib/api";
import { AUTHED_NAV_LINKS, AppNavActions } from "@/components/dashboard/app-nav";
import { EmptyPanel, ErrorPanel } from "@/components/dashboard/async-panels";
import { formatTopicLabel } from "@/components/charts/chart-theme";
import { ProblemCard } from "./_components/problem-card";
import { ProblemPreviewDialog, type ProblemPreviewSource } from "./_components/problem-preview-dialog";
import { ScrapeProblemDialog } from "./_components/scrape-problem-dialog";
import { DIFFICULTY_OPTIONS, SOURCE_OPTIONS, TOPIC_OPTIONS } from "./_lib/constants";
import type { ScrapedProblem } from "@/lib/api/types";

const PAGE_SIZE = 12;
const ALL_VALUE = "__all__";

function useDebouncedValue<T>(value: T, delayMs: number): T {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const id = setTimeout(() => setDebounced(value), delayMs);
    return () => clearTimeout(id);
  }, [value, delayMs]);
  return debounced;
}

export default function ProblemsPage() {
  const { isAuthenticated, isLoading: authLoading } = useAuth();
  const [qInput, setQInput] = useState("");
  const [topic, setTopic] = useState<string>(ALL_VALUE);
  const [source, setSource] = useState<string>(ALL_VALUE);
  const [difficulty, setDifficulty] = useState<string>(ALL_VALUE);
  const [offset, setOffset] = useState(0);
  const [previewSource, setPreviewSource] = useState<ProblemPreviewSource | null>(null);

  const debouncedQ = useDebouncedValue(qInput, 350);

  useEffect(() => {
    setOffset(0);
  }, [debouncedQ, topic, source, difficulty]);

  const params = {
    q: debouncedQ || undefined,
    topic: topic === ALL_VALUE ? undefined : topic,
    source: source === ALL_VALUE ? undefined : source,
    difficulty_label: difficulty === ALL_VALUE ? undefined : difficulty,
    limit: PAGE_SIZE,
    offset,
  };

  const searchQuery = useQuery({
    queryKey: ["problems", "search", params],
    queryFn: () => searchProblems(params),
    placeholderData: (prev) => prev,
    enabled: isAuthenticated === true,
  });

  const hasActiveFilters = Boolean(debouncedQ || topic !== ALL_VALUE || source !== ALL_VALUE || difficulty !== ALL_VALUE);

  function clearFilters() {
    setQInput("");
    setTopic(ALL_VALUE);
    setSource(ALL_VALUE);
    setDifficulty(ALL_VALUE);
  }

  function handleScraped(scraped: ScrapedProblem) {
    const matchedId = searchQuery.data?.problems.find((p) => p.url === scraped.url)?.id;
    setPreviewSource({ kind: "scraped", scraped, matchedId });
  }

  const total = searchQuery.data?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));
  const currentPage = Math.floor(offset / PAGE_SIZE) + 1;

  if (!authLoading && !isAuthenticated) {
    return (
      <div className="flex min-h-full flex-col bg-tv-bg">
        <TopNav links={AUTHED_NAV_LINKS} actions={<AppNavActions />} />
        <PageContainer className="flex flex-1 items-center justify-center">
          <EmptyPanel
            title="Sign in to browse problems"
            description="Search the problem bank, filter by topic or difficulty, and scrape new problems by URL once you're signed in."
            action={
              <Button render={<Link href="/signin" />} size="sm">
                Sign in
              </Button>
            }
          />
        </PageContainer>
        <Footer />
      </div>
    );
  }

  return (
    <div className="flex min-h-full flex-col bg-tv-bg">
      <TopNav links={AUTHED_NAV_LINKS} actions={<AppNavActions />} />
      <PageContainer className="flex flex-1 flex-col gap-6">
        <div className="flex flex-wrap items-end justify-between gap-4">
          <div>
            <p className="font-mono text-sm text-tv-cyan uppercase">Problems</p>
            <h1 className="font-display text-h1 font-bold text-tv-text-hi">Problem bank</h1>
          </div>
          <ScrapeProblemDialog
            trigger={
              <Button variant="outline-cyan">
                <LinkIcon />
                Scrape by URL
              </Button>
            }
            onScraped={handleScraped}
          />
        </div>

        {/* Filters — one row above everything they scope, per dataviz's interaction rules. */}
        <div className="glass-panel flex flex-wrap items-center gap-3 rounded-tv-card border border-tv-border px-4 py-3">
          <div className="relative min-w-[200px] flex-1">
            <SearchIcon className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-tv-text-body" />
            <Input
              value={qInput}
              onChange={(e) => setQInput(e.target.value)}
              placeholder="Search problems…"
              className="pl-8"
              aria-label="Search problems"
            />
          </div>

          <Select value={topic} onValueChange={(v) => setTopic(v ?? ALL_VALUE)}>
            <SelectTrigger className="w-[160px]" aria-label="Filter by topic">
              <SelectValue placeholder="Topic" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={ALL_VALUE}>All topics</SelectItem>
              {TOPIC_OPTIONS.map((t) => (
                <SelectItem key={t} value={t}>
                  {formatTopicLabel(t)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          <Select value={source} onValueChange={(v) => setSource(v ?? ALL_VALUE)}>
            <SelectTrigger className="w-[140px]" aria-label="Filter by source">
              <SelectValue placeholder="Source" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={ALL_VALUE}>All sources</SelectItem>
              {SOURCE_OPTIONS.map((s) => (
                <SelectItem key={s} value={s} className="capitalize">
                  {s}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          <Select value={difficulty} onValueChange={(v) => setDifficulty(v ?? ALL_VALUE)}>
            <SelectTrigger className="w-[140px]" aria-label="Filter by difficulty">
              <SelectValue placeholder="Difficulty" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={ALL_VALUE}>All difficulties</SelectItem>
              {DIFFICULTY_OPTIONS.map((d) => (
                <SelectItem key={d} value={d} className="capitalize">
                  {d}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          {hasActiveFilters && (
            <Button variant="ghost" size="sm" onClick={clearFilters}>
              <XIcon />
              Clear
            </Button>
          )}
        </div>

        {searchQuery.isPending ? (
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
            {Array.from({ length: 6 }).map((_, i) => (
              <Skeleton key={i} className="h-40 w-full rounded-tv-card" />
            ))}
          </div>
        ) : searchQuery.isError ? (
          <ErrorPanel error={searchQuery.error} onRetry={() => searchQuery.refetch()} />
        ) : searchQuery.data.problems.length === 0 ? (
          <EmptyPanel
            title={hasActiveFilters ? "No problems match those filters" : "No problems found"}
            description={
              hasActiveFilters
                ? "Try loosening a filter, or scrape a specific problem by URL instead."
                : "Scrape a problem by URL to add one to preview."
            }
            action={
              hasActiveFilters ? (
                <Button variant="outline-cyan" size="sm" onClick={clearFilters}>
                  Clear filters
                </Button>
              ) : undefined
            }
          />
        ) : (
          <>
            <p className="font-mono text-xs text-tv-text-body">
              {total.toLocaleString()} problem{total === 1 ? "" : "s"}
            </p>
            <div
              className={`grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3 ${searchQuery.isPlaceholderData ? "opacity-60" : ""}`}
            >
              {searchQuery.data.problems.map((problem) => (
                <ProblemCard
                  key={problem.id}
                  problem={problem}
                  onPreview={() => setPreviewSource({ kind: "problem", problem })}
                />
              ))}
            </div>

            {totalPages > 1 && (
              <div className="flex items-center justify-between pt-2">
                <Button variant="outline" size="sm" disabled={offset === 0} onClick={() => setOffset((o) => Math.max(0, o - PAGE_SIZE))}>
                  <ChevronLeftIcon />
                  Previous
                </Button>
                <span className="font-mono text-xs text-tv-text-body">
                  Page {currentPage} of {totalPages}
                </span>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={offset + PAGE_SIZE >= total}
                  onClick={() => setOffset((o) => o + PAGE_SIZE)}
                >
                  Next
                  <ChevronRightIcon />
                </Button>
              </div>
            )}
          </>
        )}
      </PageContainer>
      <Footer />

      <ProblemPreviewDialog source={previewSource} onOpenChange={(open) => !open && setPreviewSource(null)} />
    </div>
  );
}
