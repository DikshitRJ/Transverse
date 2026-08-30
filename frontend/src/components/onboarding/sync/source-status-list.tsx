import { Loader2, X } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import type { SyncSource } from "@/components/onboarding/sync/types";
import type { EvidenceStatus } from "@/lib/api/types";

// "default"/"secondary" resolve to solid `--primary` (== --tv-cyan) per the
// shadcn token remap in globals.css — too close to "success" to tell apart
// at a glance, so in-flight states use the neutral "outline" variant
// instead and let "done" (success, cyan) / "failed" (error, rose) be the
// two visually distinct end states.
const STATUS_COPY: Record<EvidenceStatus, { label: string; variant: "outline" | "success" | "error" | "locked" }> = {
  pending: { label: "Pending", variant: "locked" },
  fetching: { label: "Fetching", variant: "outline" },
  processing: { label: "Processing", variant: "outline" },
  done: { label: "Done", variant: "success" },
  failed: { label: "Failed", variant: "error" },
  purged: { label: "Purged", variant: "locked" },
};

const KIND_LABEL: Record<SyncSource["kind"], string> = {
  resume: "Resume",
  codebase: "Codebase",
  github: "GitHub",
  leetcode: "LeetCode",
  codeforces: "Codeforces",
};

/**
 * Per-source status list (pending -> fetching -> processing -> done/failed).
 * See `use-evidence-sync.ts` for why "done" is a best-effort signal rather
 * than a confirmed read of backend state.
 */
export function SourceStatusList({
  sources,
  onRemove,
}: {
  sources: SyncSource[];
  onRemove: (id: string) => void;
}) {
  if (sources.length === 0) {
    return (
      <p className="rounded-tv-card border border-tv-border bg-tv-surface px-4 py-6 text-center font-mono text-sm text-tv-text-body">
        No sources added yet — connect a platform or upload a file above.
      </p>
    );
  }

  return (
    <ul className="flex flex-col gap-2">
      {sources.map((source) => {
        const copy = STATUS_COPY[source.status];
        const isInFlight = source.status === "pending" || source.status === "fetching" || source.status === "processing";
        return (
          <li
            key={source.id}
            className="flex items-center justify-between gap-3 rounded-tv-btn border border-tv-border bg-tv-surface px-4 py-3"
          >
            <div className="flex min-w-0 flex-col">
              <span className="truncate font-mono text-sm text-tv-text-hi">{source.label}</span>
              <span className="font-body text-xs text-tv-text-body">{KIND_LABEL[source.kind]}</span>
              {source.status === "failed" && source.errorMessage ? (
                <span className="mt-1 font-body text-xs text-tv-rose">{source.errorMessage}</span>
              ) : null}
            </div>
            <div className="flex shrink-0 items-center gap-2">
              <Badge variant={copy.variant} className="gap-1">
                {isInFlight ? <Loader2 className="size-3 animate-spin motion-reduce:animate-none" /> : null}
                {copy.label}
              </Badge>
              <button
                type="button"
                onClick={() => onRemove(source.id)}
                aria-label={`Remove ${source.label}`}
                className="rounded-tv-chip p-1 text-tv-text-body transition-colors hover:bg-tv-surface-2 hover:text-tv-text-hi"
              >
                <X className="size-3.5" />
              </button>
            </div>
          </li>
        );
      })}
    </ul>
  );
}
