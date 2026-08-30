import Image from "next/image";
import Link from "next/link";
import type { ReactNode } from "react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";

export interface TopNavLink {
  label: string;
  href: string;
}

const DEFAULT_LINKS: TopNavLink[] = [
  { label: "Analyze", href: "/#analyze" },
  { label: "Quiz", href: "/onboarding/quiz" },
];

export interface TopNavProps {
  links?: TopNavLink[];
  /** Right-side slot. Defaults to a primary "Get Started" CTA — pass your own for authenticated chrome (avatar menu, etc). */
  actions?: ReactNode;
  className?: string;
}

/**
 * App shell top nav — matches Figma `61:110`: glass panel, `rgba(0,255,255,0.5)`
 * border + glow, Space Grotesk logo with cyan text-glow, gradient
 * "GET STARTED" CTA. Reused across every route (not just landing) — pass
 * `links`/`actions` to adapt it to signed-in chrome.
 */
export function TopNav({ links = DEFAULT_LINKS, actions, className }: TopNavProps) {
  return (
    <header
      className={cn(
        "glass-panel sticky top-0 z-50 flex w-full items-center justify-between border-tv-border-nav px-6 py-4 shadow-[0_0_15px_0_rgba(0,255,255,0.1)]",
        className,
      )}
    >
      <Link href="/" className="flex shrink-0 items-center gap-3">
        <Image
          src="/figma/byte-mascot-nav.png"
          alt="Byte the Beaver"
          width={40}
          height={36}
          className="h-9 w-10 object-contain"
          priority
        />
        <span className="glow-text-cyan font-display text-[28px] leading-8 font-bold tracking-[-1.2px] text-tv-text-hi uppercase">
          Transverse
        </span>
      </Link>

      <nav className="hidden items-center gap-8 md:flex">
        {links.map((link) => (
          <Link
            key={link.href}
            href={link.href}
            className="font-body px-2 py-1 text-sm font-semibold text-tv-text-nav uppercase transition-colors hover:text-tv-text-hi"
          >
            {link.label}
          </Link>
        ))}
      </nav>

      <div className="flex shrink-0 items-center gap-4">
        {actions ?? (
          <Button render={<Link href="/onboarding" />}>Get Started</Button>
        )}
      </div>
    </header>
  );
}
