import { Suspense } from "react";
import type { Metadata } from "next";
import Link from "next/link";
import { BrandHeader } from "@/components/auth/brand-header";
import { AuthForm } from "@/components/auth/auth-form";
import { ByteAvatar } from "@/components/onboarding/byte-avatar";

export const metadata: Metadata = {
  title: "Sign in — Transverse",
};

/**
 * Net-new (plan.md route #2) — sign in page with email & password authentication.
 */
export default function SignInPage() {
  return (
    <div className="flex min-h-full flex-col bg-tv-bg">
      <BrandHeader />

      <main className="flex flex-1 items-center justify-center px-6 py-12">
        <div className="glass-panel glow-card-cyan w-full max-w-[440px] rounded-tv-card p-8">
          <div className="flex flex-col items-center text-center">
            <ByteAvatar className="size-14 border border-tv-border-cyan bg-tv-surface-deep" />

            <h1 className="glow-text-cyan mt-5 font-display text-h1 font-bold tracking-[-1.2px] text-tv-text-hi uppercase">
              Welcome
            </h1>
            <p className="mt-1.5 font-mono text-sm text-tv-text-body">
              Sign in to calibrate your learning path and continue mastering DSA.
            </p>
          </div>

          <Suspense fallback={<div className="h-64 flex items-center justify-center font-mono text-xs text-tv-text-body">Loading sign in form...</div>}>
            <AuthForm className="mt-6" />
          </Suspense>

          <p className="mt-6 text-center font-body text-xs text-tv-text-body">
            By continuing you agree to Transverse&apos;s{" "}
            <Link href="/terms" className="text-tv-cyan hover:underline">Terms</Link>
            {" "}and{" "}
            <Link href="/privacy" className="text-tv-cyan hover:underline">Privacy Policy</Link>.
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
