import type { Metadata } from "next";
import Link from "next/link";
import { TopNav } from "@/components/shell/top-nav";
import { Footer } from "@/components/shell/footer";
import { PageContainer } from "@/components/shell/page-container";

export const metadata: Metadata = {
  title: "Privacy Policy — Transverse",
  description: "Privacy policy and zero-cloud persistence guarantees of the Transverse platform.",
};

export default function PrivacyPage() {
  return (
    <div className="flex min-h-full flex-col bg-tv-bg">
      <TopNav />
      <main className="flex flex-1 flex-col py-12 md:py-16">
        <PageContainer>
          <div className="glass-panel glow-card-cyan rounded-tv-card p-8 md:p-12">
            <div className="flex flex-col gap-2 border-b border-tv-border-muted pb-6">
              <span className="font-mono text-xs text-tv-cyan uppercase tracking-widest">
                Data & Privacy
              </span>
              <h1 className="glow-text-cyan font-display text-h1 font-bold tracking-tight text-tv-text-hi uppercase">
                Privacy Policy
              </h1>
              <p className="font-mono text-xs text-tv-text-body">
                Last updated: August 30, 2026
              </p>
            </div>

            <div className="mt-8 space-y-8 text-sm text-tv-text-body font-body leading-relaxed">
              {/* Highlight callout box */}
              <div className="rounded-lg border border-tv-cyan/30 bg-tv-cyan/5 p-4 text-xs font-mono text-tv-text-hi leading-normal">
                <span className="font-bold text-tv-cyan">🔒 Privacy by Default Architecture:</span> Transverse never persists raw resumes, scraped source repositories, or unanonymized code files to permanent storage. Raw artifact uploads are processed in volatile memory / transient buckets and deleted via automatic deferred cleanup handlers once numeric skill signals are extracted.
              </div>

              <section className="space-y-3">
                <h2 className="font-display text-lg font-bold text-tv-text-hi uppercase tracking-wide">
                  1. Information We Collect
                </h2>
                <p>We collect only the essential information necessary to deliver personalized adaptive tutoring:</p>
                <ul className="list-disc pl-5 space-y-1.5 text-tv-text-hi/90 font-mono text-xs">
                  <li><strong className="text-tv-cyan">Account Data:</strong> Email address, hashed password, and username.</li>
                  <li><strong className="text-tv-cyan">Performance Telemetry:</strong> Code execution submissions, solve velocities, testcase verdicts, error patterns, and psychometric ratings (IRT &theta;, Glicko-2).</li>
                  <li><strong className="text-tv-cyan">Public Profiles:</strong> Public repository metadata or competitive programming handles if you explicitly connect GitHub, LeetCode, or Codeforces.</li>
                </ul>
              </section>

              <section className="space-y-3">
                <h2 className="font-display text-lg font-bold text-tv-text-hi uppercase tracking-wide">
                  2. How We Use Your Data
                </h2>
                <p>Collected telemetry is utilized strictly for:</p>
                <ul className="list-disc pl-5 space-y-1 text-tv-text-hi/90 font-mono text-xs">
                  <li>Calibrating problem difficulty according to your Zone of Proximal Development.</li>
                  <li>Generating personalized roadmap checkpoints and targeted tutorial recommendations.</li>
                  <li>Providing real-time hint generation and closed-loop error remediation.</li>
                  <li>Maintaining your learning streak and historical progress analytics.</li>
                </ul>
              </section>

              <section className="space-y-3">
                <h2 className="font-display text-lg font-bold text-tv-text-hi uppercase tracking-wide">
                  3. Cookies & Local Session Storage
                </h2>
                <p>
                  Transverse uses secure, <code className="font-mono text-xs text-tv-cyan bg-tv-surface-deep px-1.5 py-0.5 rounded">HttpOnly</code>,
                  <code className="font-mono text-xs text-tv-cyan bg-tv-surface-deep px-1.5 py-0.5 rounded ml-1">SameSite=Lax</code> session
                  cookies to maintain user authentication without exposing tokens to client-side scripts. No third-party ad tracking cookies are deployed.
                </p>
              </section>

              <section className="space-y-3">
                <h2 className="font-display text-lg font-bold text-tv-text-hi uppercase tracking-wide">
                  4. Third-Party Services
                </h2>
                <p>
                  When executing code or generating AI hints, code payloads are processed in isolated, sandboxed environments
                  (Judge0 engine) and AI inference endpoints with zero data retention for training.
                </p>
              </section>

              <section className="space-y-3">
                <h2 className="font-display text-lg font-bold text-tv-text-hi uppercase tracking-wide">
                  5. Data Retention & Deletion Rights
                </h2>
                <p>
                  You retain complete ownership of your learning profile. You may request account deletion and the purge of all
                  associated psychometric history by contacting our support team at{" "}
                  <a href="mailto:privacy@transverse.local" className="text-tv-cyan hover:underline">
                    privacy@transverse.local
                  </a>.
                </p>
              </section>
            </div>

            <div className="mt-10 border-t border-tv-border-muted pt-6 flex justify-between items-center">
              <Link
                href="/terms"
                className="font-mono text-xs text-tv-cyan hover:underline flex items-center gap-1"
              >
                &larr; Read Terms of Service
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
