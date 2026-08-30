import type { Metadata } from "next";
import Link from "next/link";
import { TopNav } from "@/components/shell/top-nav";
import { Footer } from "@/components/shell/footer";
import { PageContainer } from "@/components/shell/page-container";

export const metadata: Metadata = {
  title: "Terms of Service — Transverse",
  description: "Terms and conditions governing the use of the Transverse adaptive learning platform.",
};

export default function TermsPage() {
  return (
    <div className="flex min-h-full flex-col bg-tv-bg">
      <TopNav />
      <main className="flex flex-1 flex-col py-12 md:py-16">
        <PageContainer>
          <div className="glass-panel glow-card-cyan rounded-tv-card p-8 md:p-12">
            <div className="flex flex-col gap-2 border-b border-tv-border-muted pb-6">
              <span className="font-mono text-xs text-tv-cyan uppercase tracking-widest">
                Legal & Governance
              </span>
              <h1 className="glow-text-cyan font-display text-h1 font-bold tracking-tight text-tv-text-hi uppercase">
                Terms of Service
              </h1>
              <p className="font-mono text-xs text-tv-text-body">
                Last updated: August 30, 2026
              </p>
            </div>

            <div className="mt-8 space-y-8 text-sm text-tv-text-body font-body leading-relaxed">
              <section className="space-y-3">
                <h2 className="font-display text-lg font-bold text-tv-text-hi uppercase tracking-wide">
                  1. Acceptance of Terms
                </h2>
                <p>
                  By accessing or using Transverse (&ldquo;the Platform&rdquo;, &ldquo;we&rdquo;, &ldquo;us&rdquo;, or &ldquo;our&rdquo;),
                  you agree to be bound by these Terms of Service. If you do not agree to these terms, please do not use the Platform.
                </p>
              </section>

              <section className="space-y-3">
                <h2 className="font-display text-lg font-bold text-tv-text-hi uppercase tracking-wide">
                  2. Platform Description & Adaptive Learning
                </h2>
                <p>
                  Transverse is an adaptive learning engine and competitive programming preparation platform that evaluates
                  developer capability using psychometric models (Item Response Theory and Glicko-2) to construct dynamic curriculum roadmaps.
                </p>
                <p>
                  All performance metrics, skill ratings, ability scores (&theta;), and learning roadmaps are dynamically computed
                  telemetry designed to optimize your pedagogical growth.
                </p>
              </section>

              <section className="space-y-3">
                <h2 className="font-display text-lg font-bold text-tv-text-hi uppercase tracking-wide">
                  3. User Accounts & Security
                </h2>
                <p>
                  You are responsible for maintaining the confidentiality of your account credentials (email and password)
                  and for all activities that occur under your account. You agree to notify us immediately of any unauthorized
                  use or security breach.
                </p>
              </section>

              <section className="space-y-3">
                <h2 className="font-display text-lg font-bold text-tv-text-hi uppercase tracking-wide">
                  4. Acceptable Use Policy
                </h2>
                <p>When utilizing Transverse and its integrated Judge0 execution sandboxes, you agree NOT to:</p>
                <ul className="list-disc pl-5 space-y-1 text-tv-text-hi/90 font-mono text-xs">
                  <li>Attempt to execute malicious code, denial of service attacks, or escape sandbox boundaries.</li>
                  <li>Scrape, reverse engineer, or extract proprietary heuristic models or curriculum graphs without authorization.</li>
                  <li>Use automated bots or scripts to manipulate rating systems or practice session analytics.</li>
                  <li>Share abusive, harmful, or unlawful content through profile or submission metadata.</li>
                </ul>
              </section>

              <section className="space-y-3">
                <h2 className="font-display text-lg font-bold text-tv-text-hi uppercase tracking-wide">
                  5. Intellectual Property
                </h2>
                <p>
                  The algorithms, UI design, mascot assets, adaptive graph structures, and codebase of Transverse are the
                  exclusive intellectual property of Transverse. Practice problems curated from open competitive programming
                  platforms remain the property of their respective creators and sources.
                </p>
              </section>

              <section className="space-y-3">
                <h2 className="font-display text-lg font-bold text-tv-text-hi uppercase tracking-wide">
                  6. Disclaimer of Warranties & Limitation of Liability
                </h2>
                <p>
                  The Platform is provided on an &ldquo;AS IS&rdquo; and &ldquo;AS AVAILABLE&rdquo; basis without warranties of any kind.
                  Transverse does not guarantee uninterrupted service or specific competitive programming outcomes.
                </p>
              </section>

              <section className="space-y-3">
                <h2 className="font-display text-lg font-bold text-tv-text-hi uppercase tracking-wide">
                  7. Contact Information
                </h2>
                <p>
                  For questions regarding these Terms of Service, please contact us at{" "}
                  <a href="mailto:support@transverse.local" className="text-tv-cyan hover:underline">
                    support@transverse.local
                  </a>.
                </p>
              </section>
            </div>

            <div className="mt-10 border-t border-tv-border-muted pt-6 flex justify-between items-center">
              <Link
                href="/privacy"
                className="font-mono text-xs text-tv-cyan hover:underline flex items-center gap-1"
              >
                Read Privacy Policy &rarr;
              </Link>
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
