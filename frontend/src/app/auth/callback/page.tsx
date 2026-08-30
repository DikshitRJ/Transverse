"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { BrandHeader } from "@/components/auth/brand-header";
import { postSignInDestination } from "@/components/auth/user-status";
import { useAuth } from "@/components/providers/auth-provider";
import { getMe } from "@/lib/api/endpoints";
import { Button } from "@/components/ui/button";

const OAUTH_ERROR_COPY: Record<string, string> = {
  access_denied: "You declined the sign-in request, so we couldn't continue.",
  server_error: "The sign-in provider had a problem completing this request.",
  temporarily_unavailable: "The sign-in provider is temporarily unavailable. Please try again shortly.",
};

type CallbackState = { status: "pending" } | { status: "error"; message: string };

/**
 * Plan.md route #3. Backend contract (documented as an *assumption* in
 * FOUNDATION.md §7, not yet confirmed against KEYSTONE's landed
 * `auth_handler.go`): the OAuth callback 302s the browser straight here
 * with `access_token`/`refresh_token`/`expires_in` as query params —
 * verified as what the mock `GET /api/v1/auth/oauth/:provider/redirect`
 * handler actually does. If the real backend instead lands here with an
 * `error`/`error_description` pair (denied consent, provider failure) or
 * simply omits the token params, that's handled below as a real UI state,
 * not a crash.
 */
export default function AuthCallbackPage() {
  const router = useRouter();
  const { completeOAuthCallback } = useAuth();
  const [state, setState] = useState<CallbackState>({ status: "pending" });
  const ranRef = useRef(false);

  useEffect(() => {
    if (ranRef.current) return;
    ranRef.current = true;

    const params = new URLSearchParams(window.location.search);
    const oauthError = params.get("error");
    if (oauthError) {
      const description = params.get("error_description");
      setState({
        status: "error",
        message:
          description ||
          OAUTH_ERROR_COPY[oauthError] ||
          `Sign-in failed (${oauthError}).`,
      });
      return;
    }

    const accessToken = params.get("access_token");
    const refreshToken = params.get("refresh_token");
    if (!accessToken || !refreshToken) {
      setState({
        status: "error",
        message: "This sign-in link is missing required information. Please try again.",
      });
      return;
    }

    (async () => {
      try {
        await completeOAuthCallback({ accessToken, refreshToken });
        // completeOAuthCallback() already fetches /auth/me into context
        // state, but that update lands asynchronously via React state —
        // fetch it directly here too so the redirect decision below is
        // never racing a stale render.
        const user = await getMe();
        router.replace(postSignInDestination(user));
      } catch {
        setState({
          status: "error",
          message: "We couldn't finish signing you in. Please try again.",
        });
      }
    })();
  }, [completeOAuthCallback, router]);

  return (
    <div className="flex min-h-full flex-col bg-tv-bg">
      <BrandHeader />
      <main className="flex flex-1 items-center justify-center px-6 py-16">
        <div className="glass-panel w-full max-w-[440px] rounded-tv-card p-8 text-center">
          {state.status === "pending" ? (
            <>
              <div
                aria-hidden
                className="mx-auto size-10 animate-spin rounded-full border-2 border-tv-border-cyan border-t-tv-cyan motion-reduce:animate-none"
              />
              <h1 className="mt-6 font-display text-h2 font-bold text-tv-text-hi uppercase">
                Signing you in&hellip;
              </h1>
              <p className="mt-2 font-mono text-sm text-tv-text-body" role="status" aria-live="polite">
                Completing the handshake with your provider.
              </p>
            </>
          ) : (
            <>
              <h1 className="font-display text-h2 font-bold text-tv-rose uppercase">
                Sign-in failed
              </h1>
              <p className="mt-2 font-mono text-sm text-tv-text-body" role="alert">
                {state.message}
              </p>
              <Button render={<Link href="/signin" />} className="mt-6 w-full normal-case">
                Back to sign in
              </Button>
            </>
          )}
        </div>
      </main>
    </div>
  );
}
