/**
 * In-memory store backing the 5 evidence routes (KEYSTONE mounts these per
 * plan.md §9.1 — not on `main` as of this writing). Upload confirmation and
 * connector syncs "complete" a few seconds after being started, purely for
 * UI dev realism (spinner -> done).
 */
import type { EvidenceKind, EvidenceStatus } from "@/lib/api/types";

interface MockEvidenceSource {
  id: string;
  userId: string;
  kind: EvidenceKind;
  status: EvidenceStatus;
}

const store = new Map<string, MockEvidenceSource>();
let counter = 0;

function nextId(): string {
  counter += 1;
  return `evidence-mock-${counter}`;
}

export function createUploadEvidence(
  userId: string,
  kind: EvidenceKind,
): { evidenceId: string; uploadUrl: string } {
  const id = nextId();
  store.set(id, { id, userId, kind, status: "pending" });
  const uploadUrl = `https://mock-object-store.example.com/upload/${id}`;
  return { evidenceId: id, uploadUrl };
}

export function confirmEvidenceUpload(evidenceId: string): MockEvidenceSource | undefined {
  const entry = store.get(evidenceId);
  if (!entry) return undefined;
  entry.status = "processing";
  setTimeout(() => {
    const e = store.get(evidenceId);
    if (e) e.status = "done";
  }, 3000);
  return entry;
}

export function createConnectorEvidence(userId: string, kind: EvidenceKind): string {
  const id = nextId();
  store.set(id, { id, userId, kind, status: "fetching" });
  setTimeout(() => {
    const e = store.get(id);
    if (e) e.status = "processing";
  }, 1500);
  setTimeout(() => {
    const e = store.get(id);
    if (e) e.status = "done";
  }, 4000);
  return id;
}

export function getEvidence(evidenceId: string): MockEvidenceSource | undefined {
  return store.get(evidenceId);
}
