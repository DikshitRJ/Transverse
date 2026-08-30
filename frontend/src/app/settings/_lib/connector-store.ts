/**
 * Local, best-effort record of which evidence connectors the user has
 * submitted a sync request for. There is no `GET /evidence` (list) route —
 * only the write endpoints (`POST /evidence/{github|leetcode|codeforces}`,
 * each 202 `{evidence_id, status: "processing_queued"}`) — so the backend
 * has no way for this page to ask "what's actually connected right now."
 * This is `localStorage`-backed and explicitly labeled "last known" in the
 * UI rather than presented as authoritative connector state.
 */

export type ConnectorKind = "github" | "leetcode" | "codeforces";

export interface ConnectorRecord {
  identifier: string;
  evidenceId: string;
  queuedAt: string;
}

const STORAGE_KEY = "tv.settings.connectors";

type StoredMap = Partial<Record<ConnectorKind, ConnectorRecord>>;

function readAll(): StoredMap {
  if (typeof window === "undefined") return {};
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return {};
    const parsed: unknown = JSON.parse(raw);
    if (typeof parsed !== "object" || parsed === null) return {};
    return parsed as StoredMap;
  } catch {
    return {};
  }
}

export function getConnectorRecords(): StoredMap {
  return readAll();
}

export function saveConnectorRecord(kind: ConnectorKind, record: ConnectorRecord): void {
  if (typeof window === "undefined") return;
  const all = readAll();
  all[kind] = record;
  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(all));
  } catch {
    // best-effort only — connector state just won't persist across reloads.
  }
}

export function clearConnectorRecord(kind: ConnectorKind): void {
  if (typeof window === "undefined") return;
  const all = readAll();
  delete all[kind];
  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(all));
  } catch {
    // ignore
  }
}
