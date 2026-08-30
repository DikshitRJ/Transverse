import type { Metadata } from "next";
import Link from "next/link";
import { BrandHeader } from "@/components/auth/brand-header";
import { OAuthButtons } from "@/components/auth/oauth-buttons";
import { ByteAvatar } from "@/components/onboarding/byte-avatar";

export const metadata: Metadata = {
  title: "Sign in — Transverse",
};

/**
 * Net-new (plan.md route #2) — no Figma frame exists for this screen.
 * Composed from the frozen token set in the same visual language as
 * `/onboarding`: dark ground, glass card, cyan primary, Byte present but
 * understated (just the header lockup + a small avatar on the card, not a
 * hero mascot). Real OAuth only — no dev-bypass affordance (plan.md §9.3).
 */
export default function SignInPage() {
  return (
    <div className="flex min-h-full flex-col bg-tv-bg">
      <BrandHeader />

      <main className="flex flex-1 items-center justify-center px-6 py-16">
        <div className="glass-panel glow-card-cyan w-full max-w-[440px] rounded-tv-card p-8">
          <div className="flex flex-col items-center text-center">
            <ByteAvatar className="size-14 border border-tv-border-cyan bg-tv-surface-deep" />

            <h1 className="glow-text-cyan mt-6 font-display text-h1 font-bold tracking-[-1.2px] text-tv-text-hi uppercase">
              Sign in
            </h1>
            <p className="mt-2 font-mono text-sm text-tv-text-body">
              Sign in to calibrate your learning path and pick up where you
              left off.
            </p>
          </div>

          <OAuthButtons className="mt-8" />

          <p className="mt-6 text-center font-body text-xs text-tv-text-body">
            By continuing you agree to Transverse&apos;s Terms and Privacy
            Policy. We only ever request read access to your public profile.
          </p>
        </div>
      </main>

      <footer className="px-6 pb-10 text-center">
        <Link
          href="/"
          className="font-mono text-xs text-tv-text-body underline-offset-4 hover:text-tv-cyan hover:underline"
        >
          &larr; Back to Transverse
        </Link>
      </footer>
    </div>
  );
}
