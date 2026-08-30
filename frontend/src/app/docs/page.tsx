import type { Metadata } from "next";
import Link from "next/link";
import { TopNav } from "@/components/shell/top-nav";
import { Footer } from "@/components/shell/footer";
import { PageContainer } from "@/components/shell/page-container";
import { BookOpen, Cpu, Target, Shield, Compass, Sparkles } from "lucide-react";

export const metadata: Metadata = {
  title: "Documentation — Transverse",
  description: "Comprehensive guide to the Transverse adaptive heuristic engine, roadmap mechanics, and scoring algorithms.",
};

export default function DocsPage() {
  return (
    <div className="flex min-h-full flex-col bg-tv-bg">
      <TopNav />
      <main className="flex flex-1 flex-col py-12 md:py-16">
        <PageContainer>
          <div className="glass-panel glow-card-cyan rounded-tv-card p-8 md:p-12">
            <div className="flex flex-col gap-2 border-b border-tv-border-muted pb-6">
              <span className="font-mono text-xs text-tv-cyan uppercase tracking-widest">
                Platform Architecture & Guide
              </span>
              <h1 className="glow-text-cyan font-display text-h1 font-bold tracking-tight text-tv-text-hi uppercase">
                Transverse Documentation
              </h1>
              <p className="font-mono text-xs text-tv-text-body">
                Version 2.0 • Adaptive Heuristic Engine
              </p>
            </div>

            <div className="mt-8 space-y-10 text-sm text-tv-text-body font-body leading-relaxed">
              {/* Grid of Key Features */}
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4 font-mono text-xs">
                <div className="rounded-lg border border-tv-cyan/30 bg-tv-surface-deep/80 p-4 space-y-2">
                  <div className="flex items-center gap-2 text-tv-cyan font-bold">
                    <Target className="size-4" />
                    <span>Zone of Proximal Dev</span>
                  </div>
                  <p className="text-tv-text-body leading-normal">
                    Problems are served within a tightly bounded difficulty window matching your current ability parameter (&theta;).
                  </p>
                </div>

                <div className="rounded-lg border border-tv-cyan/30 bg-tv-surface-deep/80 p-4 space-y-2">
                  <div className="flex items-center gap-2 text-tv-cyan font-bold">
                    <Cpu className="size-4" />
                    <span>Deterministic Glicko-2</span>
                  </div>
                  <p className="text-tv-text-body leading-normal">
                    Real-time rating, rating deviation (RD), and volatility updates ensure statistical rigor without probabilistic hallucination.
                  </p>
                </div>

                <div className="rounded-lg border border-tv-cyan/30 bg-tv-surface-deep/80 p-4 space-y-2">
                  <div className="flex items-center gap-2 text-tv-cyan font-bold">
                    <Compass className="size-4" />
                    <span>Dynamic DAG Roadmaps</span>
                  </div>
                  <p className="text-tv-text-body leading-normal">
                    Nodes unlock sequentially based on prerequisite mastery, with Duolingo-style &ldquo;test-out&rdquo; bypass mechanics.
                  </p>
                </div>
              </div>

              <section className="space-y-3">
                <h2 className="font-display text-lg font-bold text-tv-text-hi uppercase tracking-wide flex items-center gap-2">
                  <Sparkles className="size-5 text-tv-cyan" />
                  1. Adaptive Practice Loop
                </h2>
                <p>
                  Transverse assesses every code submission against automated testcases using the high-performance Judge0 runtime.
                  When you solve or fail a problem, the scoring engine recalibrates your latent ability (&theta;) and psychometric DNA:
                </p>
                <ul className="list-disc pl-5 space-y-1.5 text-tv-text-hi/90 font-mono text-xs">
                  <li><strong className="text-tv-cyan">Solve Velocity:</strong> Tracks your time-to-first-pass relative to problem historical averages.</li>
                  <li><strong className="text-tv-cyan">Carelessness Penalty:</strong> Penalizes trivial compilation errors or edge-case oversights to foster defensive programming habits.</li>
                  <li><strong className="text-tv-cyan">Closed-Loop Remediation:</strong> 3 consecutive failures automatically trigger difficulty attenuation and structured concept reviews.</li>
                </ul>
              </section>

              <section className="space-y-3">
                <h2 className="font-display text-lg font-bold text-tv-text-hi uppercase tracking-wide flex items-center gap-2">
                  <BookOpen className="size-5 text-tv-cyan" />
                  2. Roadmap Navigation & Test-Outs
                </h2>
                <p>
                  The curriculum is organized as a directed acyclic graph (DAG) across key algorithmic topics: Foundations,
                  Arrays & Hashing, Two Pointers, Sliding Window, Stack & Queues, Binary Search, Trees, Graphs, and Dynamic Programming.
                </p>
                <p>
                  Experienced developers can bypass preliminary modules by taking an accelerated &ldquo;Test-Out Quiz&rdquo;
                  proving topic mastery without completing every prerequisite practice node.
                </p>
              </section>

              <section className="space-y-3">
                <h2 className="font-display text-lg font-bold text-tv-text-hi uppercase tracking-wide flex items-center gap-2">
                  <Shield className="size-5 text-tv-cyan" />
                  3. Privacy & Zero-Persistence Architecture
                </h2>
                <p>
                  Transverse ensures privacy by design. External profiles connected during onboarding are analyzed transiently.
                  Raw files are immediately scrubbed post-signal extraction, persisting only anonymized skill vector embeddings.
                </p>
              </section>
            </div>

            <div className="mt-10 border-t border-tv-border-muted pt-6 flex flex-wrap justify-between items-center gap-4">
              <div className="flex items-center gap-4 font-mono text-xs">
                <Link
                  href="/signin"
                  className="text-tv-cyan hover:underline flex items-center gap-1"
                >
                  Sign In to Platform &rarr;
                </Link>
                <a
                  href="https://github.com/dikshitrj/transverse"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-tv-text-body hover:text-tv-cyan transition-colors flex items-center gap-1"
                >
                  GitHub Repository &rarr;
                </a>
              </div>
              <Link
                href="/"
                className="font-mono text-xs text-tv-text-body hover:text-tv-text-hi"
              >
                &larr; Return to Home
              </Link>
            </div>
          </div>
        </PageContainer>
      </main>
      <Footer />
    </div>
  );
}
