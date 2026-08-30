"use client";

import { useQuery } from "@tanstack/react-query";
import { Skeleton } from "@/components/ui/skeleton";
import { getPracticeTopics } from "@/lib/api/endpoints";
import { cn } from "@/lib/utils";
import { formatMasteryScore, topicLabel } from "./format";

export interface TopicScopePickerProps {
  selected: string[];
  onToggle: (topic: string) => void;
  className?: string;
}

/** Multi-select topic scope for POST /practice/start, sourced from GET /practice/topics. */
export function TopicScopePicker({ selected, onToggle, className }: TopicScopePickerProps) {
  const { data, isLoading, isError } = useQuery({
    queryKey: ["practice", "topics"],
    queryFn: getPracticeTopics,
  });

  if (isLoading) {
    return (
      <div className={cn("flex flex-wrap gap-2", className)} aria-busy="true">
        {Array.from({ length: 8 }).map((_, i) => (
          <Skeleton key={i} className="h-8 w-28" />
        ))}
      </div>
    );
  }

  if (isError || !data || data.topics.length === 0) {
    return (
      <p className={cn("font-mono text-xs text-tv-text-body", className)}>
        Topic list unavailable — leaving scope empty practices across every topic.
      </p>
    );
  }

  return (
    <div className={cn("flex flex-wrap gap-2", className)} role="group" aria-label="Topic scope">
      {data.topics.map((t) => {
        const active = selected.includes(t.topic);
        return (
          <button
            key={t.topic}
            type="button"
            onClick={() => onToggle(t.topic)}
            aria-pressed={active}
            className={cn(
              "rounded-tv-btn border px-3 py-1.5 text-left font-mono text-xs transition-colors",
              active
                ? "border-tv-cyan bg-tv-cyan/10 text-tv-cyan glow-text-cyan"
                : "border-tv-border bg-tv-surface text-tv-text-body hover:border-tv-border-cyan hover:text-tv-text-hi",
            )}
          >
            <span className="block">{topicLabel(t.topic)}</span>
            <span className="block text-[10px] opacity-70">
              {formatMasteryScore(t.mastery_score)} mastery
            </span>
          </button>
        );
      })}
    </div>
  );
}
