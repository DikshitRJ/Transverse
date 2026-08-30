import type { EvidenceStatus } from "@/lib/api/types";

export type SyncSourceKind = "resume" | "codebase" | "github" | "leetcode" | "codeforces";

export const CONNECTOR_KINDS = ["github", "leetcode", "codeforces"] as const;
export type ConnectorKind = (typeof CONNECTOR_KINDS)[number];

export const UPLOAD_KINDS = ["resume", "codebase"] as const;
export type UploadKind = (typeof UPLOAD_KINDS)[number];

/**
 * Client-tracked view of one evidence source. Statuses reuse the real
 * `EvidenceStatus` union from `lib/api/types.ts` (`backend/internal/models/
 * evidence.go`) — see `use-evidence-sync.ts`'s doc comment for why "done"
 * is necessarily a best-effort inference rather than a confirmed read:
 * none of the 5 evidence endpoints return a status-polling id or path.
 */
export interface SyncSource {
  id: string;
  kind: SyncSourceKind;
  label: string;
  status: EvidenceStatus;
  evidenceId?: string;
  errorMessage?: string;
}
