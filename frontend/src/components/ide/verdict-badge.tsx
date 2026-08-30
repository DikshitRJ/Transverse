import type { VariantProps } from "class-variance-authority";
import { Badge, badgeVariants } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

type BadgeVariant = VariantProps<typeof badgeVariants>["variant"];

/**
 * Maps a Judge0 CE `status_id` to one of the four verdict/status Badge
 * semantics FOUNDRY already built (`success`/`warning`/`error`/`locked` —
 * see `src/components/ui/badge.tsx`). Judge0 status ids used here: 3
 * Accepted, 4 Wrong Answer, 5 Time Limit Exceeded, 6 Compilation Error,
 * 7-12 assorted runtime errors, 13 Internal Error, 14 Exec Format Error
 * (confirmed against `src/mocks/fixtures/verdicts.ts`'s scripted scenarios,
 * which follow real Judge0 CE conventions).
 */
export function judgeStatusVariant(statusId: number): BadgeVariant {
  if (statusId === 3) return "success";
  if (statusId === 5) return "warning";
  if (statusId === 4 || statusId >= 6) return "error";
  return "secondary";
}

/** True for Judge0's dedicated "Compilation Error" status (6) — the FORGE
 * brief calls this out as needing a distinct, prominent state rather than
 * just another failed test row. */
export function isCompileError(statusId: number): boolean {
  return statusId === 6;
}

export interface VerdictBadgeProps {
  statusId: number;
  statusDesc: string;
  className?: string;
}

export function VerdictBadge({ statusId, statusDesc, className }: VerdictBadgeProps) {
  return (
    <Badge variant={judgeStatusVariant(statusId)} className={cn("rounded-tv-pill", className)}>
      {statusDesc}
    </Badge>
  );
}
