"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { BrandHeader } from "@/components/auth/brand-header";
import { useAuth } from "@/components/providers/auth-provider";

/**
 * Shared shell for the whole `/onboarding/*` route group — the chooser
 * (`page.tsx`, mine), `/onboarding/sync` (mine), and `/onboarding/quiz` +
 * `/onboarding/results` (PULSE's — Next.js layouts apply to every nested
 * route, so this stays intentionally generic: page background + the
 * minimal brand header, nothing chooser-specific).
 *
 * `src/middleware.ts` already blocks the *first* unauthenticated request
 * to any `/onboarding/*` path (redirects to `/signin` before this ever
 * renders). This effect is the belt-and-suspenders case middleware can't
 * see: a session that dies *during* the visit — the in-memory access
 * token's silent refresh fails (`client.ts` on a 401, or `AuthProvider`'s
 * initial-load check) and `user` collapses to `null` without a full page
 * navigation ever happening. `isLoading` stays true only for that very
 * first check, so this never fires while a legitimate load is still in
 * flight.
 */
export default function OnboardingLayout({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, isLoading } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (!isLoading && !isAuthenticated) {
      router.replace("/signin");
    }
  }, [isLoading, isAuthenticated, router]);

  return (
    <div className="flex min-h-full flex-col bg-tv-bg-page">
      <BrandHeader />
      {children}
    </div>
  );
}
