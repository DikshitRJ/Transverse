"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { MascotBubble } from "@/components/onboarding/mascot-bubble";
import { Dropzone } from "@/components/onboarding/sync/dropzone";
import { ConnectorForm } from "@/components/onboarding/sync/connector-form";
import { SourceStatusList } from "@/components/onboarding/sync/source-status-list";
import { useEvidenceSync } from "@/components/onboarding/sync/use-evidence-sync";
import { CONNECTOR_KINDS } from "@/components/onboarding/sync/types";

/**
 * Plan.md route #5 — net-new (no Figma frame), built in the same visual
 * language as `/onboarding`. See `use-evidence-sync.ts` for the documented
 * gap between this screen's intended job-based progress model and what the
 * 5 evidence endpoints actually return.
 */
export default function EvidenceSyncPage() {
  const router = useRouter();
  const { sources, addConnector, addUpload, removeSource, summary } = useEvidenceSync();

  const canContinue = summary.total > summary.failed;

  const quote =
    summary.total === 0
      ? "Connect a platform or drop a file — I'll dig through it for signal."
      : summary.failed > 0 && summary.inFlight === 0 && summary.done === 0
        ? "That one didn't come through — mind double-checking the handle and trying again?"
        : summary.inFlight > 0
          ? "Working through it — this can take a minute for larger repos."
          : "Nice, I've got what I need. Ready when you are.";

  return (
    <main className="mx-auto flex w-full max-w-[896px] flex-1 flex-col items-center px-6 py-12 md:px-16 md:py-16">
      <h1 className="text-gradient-display glow-text-cyan text-center font-display text-[34px] leading-[34px] font-bold tracking-[-1.5px] uppercase md:text-[46px] md:leading-[46px] md:tracking-[-2.5px]">
        Sync your experience
      </h1>
      <p className="mt-4 max-w-[576px] text-center font-mono text-base font-bold text-white md:text-h3">
        Upload what you&apos;ve already built, or connect the platforms you
        already use — Transverse reads it once and folds it into your
        starting level.
      </p>

      <section className="mt-10 grid w-full grid-cols-1 gap-4 sm:grid-cols-2" aria-label="Upload evidence">
        <Dropzone
          kind="resume"
          title="Resume"
          hint="PDF or DOCX"
          accept=".pdf,.doc,.docx"
          onFile={(file) => void addUpload("resume", file)}
        />
        <Dropzone
          kind="codebase"
          title="Codebase"
          hint="Zip archive"
          accept=".zip"
          onFile={(file) => void addUpload("codebase", file)}
        />
      </section>

      <section className="mt-6 grid w-full grid-cols-1 gap-4 rounded-tv-card border border-tv-border bg-tv-surface p-6 sm:grid-cols-3" aria-label="Connect a platform">
        {CONNECTOR_KINDS.map((kind) => (
          <ConnectorForm key={kind} kind={kind} onSubmit={(value) => void addConnector(kind, value)} />
        ))}
      </section>

      <section className="mt-8 w-full" aria-label="Source status">
        <h2 className="mb-3 font-display text-sm font-bold text-tv-text-hi uppercase">Sources</h2>
        <SourceStatusList sources={sources} onRemove={removeSource} />
      </section>

      <MascotBubble quote={quote} className="mt-10" />

      <div className="mt-10 flex w-full flex-col-reverse items-center gap-3 sm:flex-row sm:justify-center">
        <Button render={<Link href="/onboarding" />} variant="ghost" className="normal-case">
          &larr; Back to options
        </Button>
        <Button
          onClick={() => router.push("/onboarding/results")}
          disabled={!canContinue}
          size="lg"
          className="normal-case"
        >
          Continue to results
        </Button>
      </div>
    </main>
  );
}
