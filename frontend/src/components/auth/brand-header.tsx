import Image from "next/image";
import Link from "next/link";
import { cn } from "@/lib/utils";

/**
 * The minimal glass header used on `/signin`, `/auth/callback`, and
 * `/onboarding` (Figma `15:10` node `15:11` — annotated "Suppressed per
 * instructions for Onboarding/Transactional" in the design file itself,
 * i.e. deliberately just the logo lockup, no nav links, no CTA). Distinct
 * from `TopNav` (`components/shell/top-nav.tsx`), which is the busy public
 * marketing header — that component's own doc comment describes it as
 * reused site-wide, but plan.md's brief for these screens explicitly asks
 * for Byte "present but understated," which the full nav (links + gradient
 * CTA) is not.
 */
export function BrandHeader({ className }: { className?: string }) {
  return (
    <header
      className={cn(
        "glass-header border-b border-tv-border-muted",
        className,
      )}
    >
      <div className="mx-auto flex max-w-[1280px] items-center px-6 py-4 md:px-16">
        <Link href="/" className="flex shrink-0 items-center gap-3">
          <Image
            src="/figma/byte-mascot-nav.png"
            alt="Byte the Beaver"
            width={47}
            height={42}
            className="h-[42px] w-[47px] object-contain"
            priority
          />
          <span className="glow-text-cyan font-display text-h2 font-bold tracking-[-1.2px] text-tv-cyan-pure uppercase">
            Transverse
          </span>
        </Link>
      </div>
    </header>
  );
}
