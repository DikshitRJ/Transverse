import type { Metadata } from "next";
import Link from "next/link";
import { TopNav } from "@/components/shell/top-nav";
import { Footer } from "@/components/shell/footer";
import { PageContainer } from "@/components/shell/page-container";

export const metadata: Metadata = {
  title: "Copyright & Legal Notice — Transverse",
  description: "Copyright notices, attribution, and open source credits for Transverse.",
};

export default function CopyrightPage() {
  return (
    <div className="flex min-h-full flex-col bg-tv-bg">
      <TopNav />
      <main className="flex flex-1 flex-col py-12 md:py-16">
        <PageContainer>
          <div className="glass-panel glow-card-cyan rounded-tv-card p-8 md:p-12">
            <div className="flex flex-col gap-2 border-b border-tv-border-muted pb-6">
              <span className="font-mono text-xs text-tv-cyan uppercase tracking-widest">
                Attribution & Licensing
              </span>
              <h1 className="glow-text-cyan font-display text-h1 font-bold tracking-tight text-tv-text-hi uppercase">
                Copyright & Legal Notice
              </h1>
              <p className="font-mono text-xs text-tv-text-body">
                © 2026 Transverse. All rights reserved.
              </p>
            </div>

            <div className="mt-8 space-y-8 text-sm text-tv-text-body font-body leading-relaxed">
              <section className="space-y-3">
                <h2 className="font-display text-lg font-bold text-tv-text-hi uppercase tracking-wide">
                  1. Ownership & Trademarks
                </h2>
                <p>
                  Transverse, the Transverse wordmark, the Byte mascot character, and associated graphical assets, interface designs,
                  and psychometric heuristic algorithms are proprietary assets and copyright of Transverse.
                </p>
              </section>

              <section className="space-y-3">
                <h2 className="font-display text-lg font-bold text-tv-text-hi uppercase tracking-wide">
                  2. Open-Source Software Credits
                </h2>
                <p>Transverse is constructed utilizing exceptional open-source technologies, including:</p>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-3 font-mono text-xs">
                  <div className="rounded-lg border border-tv-border bg-tv-surface-deep/70 p-3">
                    <span className="text-tv-cyan font-bold block">Next.js & React</span>
                    <span className="text-tv-text-body">MIT License © Vercel, Inc. & Meta Platforms, Inc.</span>
                  </div>
                  <div className="rounded-lg border border-tv-border bg-tv-surface-deep/70 p-3">
                    <span className="text-tv-cyan font-bold block">Go & Chi Router</span>
                    <span className="text-tv-text-body">BSD 3-Clause License © The Go Authors & Chi contributors</span>
                  </div>
                  <div className="rounded-lg border border-tv-border bg-tv-surface-deep/70 p-3">
                    <span className="text-tv-cyan font-bold block">PostgreSQL & pgvector</span>
                    <span className="text-tv-text-body">PostgreSQL Open License © PostgreSQL Global Development Group</span>
                  </div>
                  <div className="rounded-lg border border-tv-border bg-tv-surface-deep/70 p-3">
                    <span className="text-tv-cyan font-bold block">Judge0 API</span>
                    <span className="text-tv-text-body">GPL-3.0 License © Herman Zvonimir Došilović</span>
                  </div>
                  <div className="rounded-lg border border-tv-border bg-tv-surface-deep/70 p-3">
                    <span className="text-tv-cyan font-bold block">Tailwind CSS & Lucide Icons</span>
                    <span className="text-tv-text-body">MIT License © Tailwind Labs & Lucide Contributors</span>
                  </div>
                  <div className="rounded-lg border border-tv-border bg-tv-surface-deep/70 p-3">
                    <span className="text-tv-cyan font-bold block">Monaco Editor</span>
                    <span className="text-tv-text-body">MIT License © Microsoft Corporation</span>
                  </div>
                </div>
              </section>

              <section className="space-y-3">
                <h2 className="font-display text-lg font-bold text-tv-text-hi uppercase tracking-wide">
                  3. Content & Problem Disclaimers
                </h2>
                <p>
                  Competitive programming questions referenced from platforms such as Codeforces, AtCoder, CSES, and LeetCode
                  are indexed for educational and pedagogical practice purposes. All source attribution and platform identifiers
                  are preserved.
                </p>
              </section>
            </div>

            <div className="mt-10 border-t border-tv-border-muted pt-6 flex justify-between items-center">
              <Link
                href="/terms"
                className="font-mono text-xs text-tv-cyan hover:underline flex items-center gap-1"
              >
                &larr; View Terms of Service
              </Link>
              <Link
                href="/"
                className="font-mono text-xs text-tv-text-body hover:text-tv-text-hi"
              >
                Return to Home &rarr;
              </Link>
            </div>
          </div>
        </PageContainer>
      </main>
      <Footer />
    </div>
  );
}
