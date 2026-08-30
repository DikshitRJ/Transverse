import Image from "next/image";
import Link from "next/link";
import { cn } from "@/lib/utils";

const FOOTER_LINKS = [
  { label: "Docs", href: "/docs" },
  { label: "Terms", href: "/terms" },
  { label: "Privacy", href: "/privacy" },
  { label: "Copyright", href: "/copyright" },
  { label: "GitHub", href: "https://github.com" },
];

export interface FooterProps {
  className?: string;
}

/** Matches Figma `61:128`: glass panel, `rgba(0,255,255,0.5)` border, mascot + wordmark, link row, copyright. */
export function Footer({ className }: FooterProps) {
  return (
    <footer
      className={cn(
        "glass-panel flex w-full flex-col items-center gap-6 border-tv-border-nav px-6 py-8 shadow-[0_0_15px_0_rgba(0,255,255,0.1)] md:flex-row md:justify-between",
        className,
      )}
    >
      <Link href="/" className="flex items-center gap-3">
        <Image
          src="/figma/byte-mascot-nav.png"
          alt="Byte the Beaver"
          width={56}
          height={50}
          className="h-[50px] w-14 object-contain"
        />
        <span className="glow-text-cyan font-display text-[28px] font-bold text-tv-text-hi uppercase">
          Transverse
        </span>
      </Link>

      <nav className="flex flex-wrap items-center justify-center gap-6">
        {FOOTER_LINKS.map((link) => (
          <Link
            key={link.href}
            href={link.href}
            className="font-mono text-xs tracking-[1.2px] text-gray-400 uppercase transition-colors hover:text-tv-text-hi"
          >
            {link.label}
          </Link>
        ))}
      </nav>

      <Link
        href="/copyright"
        className="font-display text-sm font-bold tracking-[1.2px] text-tv-text-hi uppercase transition-colors hover:text-tv-cyan"
      >
        © 2026 Transverse
      </Link>
    </footer>
  );
}
