import Image from "next/image";
import Link from "next/link";
import { cn } from "@/lib/utils";

export interface QuizHeaderProps {
  eyebrow: string;
  title: string;
  subtitle?: string;
  className?: string;
}

/**
 * A lightweight funnel header for `/onboarding/quiz` and `/onboarding/results`
 * — not the full `TopNav` (its "Get Started" CTA doesn't make sense mid-funnel).
 * `src/app/onboarding/layout.tsx` is THRESHOLD's file, not built in this
 * worktree yet — this stays self-contained so both routes render correctly
 * standalone and still look right once nested under that layout later.
 */
export function QuizHeader({ eyebrow, title, subtitle, className }: QuizHeaderProps) {
  return (
    <div className={cn("flex flex-col gap-4", className)}>
      <Link href="/" className="flex w-fit items-center gap-2">
        <Image
          src="/figma/byte-mascot-chip.png"
          alt="Byte the Beaver"
          width={32}
          height={32}
          className="size-8 rounded-full object-contain"
          priority
        />
        <span className="glow-text-cyan font-display text-lg font-bold tracking-[-0.5px] text-tv-text-hi uppercase">
          Transverse
        </span>
      </Link>
      <div>
        <p className="font-mono text-xs tracking-[2px] text-tv-cyan uppercase">{eyebrow}</p>
        <h1 className="font-display text-h1 font-bold tracking-[-1px] text-tv-text-hi uppercase">
          {title}
        </h1>
        {subtitle && <p className="mt-1 max-w-2xl text-sm text-tv-text-body">{subtitle}</p>}
      </div>
    </div>
  );
}
