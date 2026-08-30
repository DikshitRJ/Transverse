import Link from "next/link";
import type { ReactNode } from "react";
import { ChevronRightIcon } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { formatPercent, formatShortDate, formatTopicLabel } from "@/components/charts/chart-theme";
import { EmptyPanel } from "./async-panels";
import { accuracyTier, summarizeSession } from "./session-utils";
import type { PracticeSession } from "@/lib/api/types";

const STATUS_BADGE: Record<string, { label: string; variant: "success" | "locked" | "secondary" }> = {
  completed: { label: "Completed", variant: "success" },
  active: { label: "In progress", variant: "secondary" },
  abandoned: { label: "Abandoned", variant: "locked" },
};

export interface RecentSessionsListProps {
  sessions: PracticeSession[];
  emptyAction?: ReactNode;
  className?: string;
}

/** Used on both `/dashboard` (compact, capped list) and `/profile` (full, paginated by the caller). */
export function RecentSessionsList({ sessions, emptyAction, className }: RecentSessionsListProps) {
  if (sessions.length === 0) {
    return (
      <EmptyPanel
        title="No practice sessions yet"
        description="Once you start an adaptive practice session, your history shows up here."
        action={emptyAction}
        className={className}
      />
    );
  }

  return (
    <ul className={className}>
      {sessions.map((session) => {
        const summary = summarizeSession(session);
        const status = STATUS_BADGE[session.status] ?? { label: session.status, variant: "secondary" as const };
        const topicLabel = summary.topics.length > 0 ? summary.topics.slice(0, 2).map(formatTopicLabel).join(", ") : "Mixed topics";
        const extraTopics = summary.topics.length - 2;

        return (
          <li key={session.id} className="border-b border-tv-border last:border-b-0">
            <Link
              href={`/practice/session/${session.id}`}
              className="flex items-center gap-4 px-1 py-3 transition-colors hover:bg-tv-surface-2/40 focus-visible:bg-tv-surface-2/40 focus-visible:outline-none"
            >
              <div className="min-w-0 flex-1">
                <div className="flex flex-wrap items-center gap-2">
                  <span className="font-mono text-xs text-tv-text-body">{formatShortDate(session.created_at)}</span>
                  <Badge variant={status.variant} className="rounded-tv-pill">
                    {status.label}
                  </Badge>
                  <span className="font-mono text-xs text-tv-text-body uppercase">{session.mode}</span>
                </div>
                <p className="mt-1 truncate font-body text-sm text-tv-text-hi">
                  {topicLabel}
                  {extraTopics > 0 && <span className="text-tv-text-body"> +{extraTopics} more</span>}
                </p>
              </div>

              <div className="flex shrink-0 items-center gap-4 text-right">
                <div>
                  <div className="font-mono text-sm text-tv-text-hi tabular-nums">{session.question_count}</div>
                  <div className="font-mono text-[10px] text-tv-text-body uppercase">Questions</div>
                </div>
                {summary.attempted > 0 && (
                  <Badge variant={accuracyTier(summary.accuracy)} className="tabular-nums">
                    {formatPercent(summary.accuracy)}
                  </Badge>
                )}
                <ChevronRightIcon className="size-4 text-tv-text-body" aria-hidden="true" />
              </div>
            </Link>
          </li>
        );
      })}
    </ul>
  );
}
