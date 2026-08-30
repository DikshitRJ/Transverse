import Image from "next/image";
import { Button } from "@/components/ui/button";
import { oauthRedirectPath } from "@/lib/api/endpoints";

/**
 * Real OAuth only (plan.md §9.3, Decision C — explicitly no dev-bypass
 * affordance). Both buttons are plain `<a href>` navigations to
 * `GET /api/v1/auth/oauth/{provider}/redirect`, never a fetch — that route
 * 307s to the real provider (or, in mock mode, short-circuits straight to
 * `/auth/callback` with working mock tokens, see `mocks/handlers.ts`).
 * `oauthRedirectPath()` is FOUNDRY's helper; it explicitly documents this
 * is not an `apiFetch` call.
 *
 * GitHub gets the real exported mark (`public/figma/github-mark.png`, the
 * same asset the onboarding "Sync Past Experiences" card uses). There is
 * no Google mark anywhere in `public/figma/` — node `15:10`/`61:*` never
 * designed a Google button, and per the design-to-code rule icons must
 * come from an exported asset, never a hand-drawn `<svg>`, so no Google
 * "G" glyph is fabricated here. The Google button ships text-only with a
 * generic `lucide-react` glyph (already the project's icon library for
 * non-brand UI, see `components/ui/dialog.tsx`/`select.tsx`) instead of a
 * fake brand mark — swap in a real Google asset once one exists.
 */
export function OAuthButtons({ className }: { className?: string }) {
  return (
    <div className={className}>
      <Button
        render={<a href={oauthRedirectPath("github")} />}
        className="w-full justify-start gap-3 normal-case"
        size="lg"
      >
        <span className="flex size-6 items-center justify-center overflow-hidden rounded-full bg-white">
          <Image
            src="/figma/github-mark.png"
            alt=""
            width={24}
            height={24}
            className="size-full object-cover"
          />
        </span>
        Continue with GitHub
      </Button>

      <Button
        render={<a href={oauthRedirectPath("google")} />}
        variant="outline-cyan"
        className="mt-3 w-full justify-start gap-3 normal-case"
        size="lg"
      >
        <GoogleGlyph className="size-6 shrink-0" />
        Continue with Google
      </Button>
    </div>
  );
}

/**
 * Neutral placeholder glyph for the Google button (see file doc comment —
 * no real Google mark ships in `public/figma/`). Deliberately abstract
 * (a plain ring, not a fake "G") so it never reads as an inaccurate brand
 * mark.
 */
function GoogleGlyph({ className }: { className?: string }) {
  return (
    <span
      aria-hidden
      className={`inline-flex items-center justify-center rounded-full border-2 border-tv-cyan ${className ?? ""}`}
    />
  );
}
