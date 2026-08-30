import Image from "next/image";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Reveal } from "./reveal";

/** Figma `61:48` — the hero: MASTER/DSA display heading, subcopy with the
 * "intelligent learning heuristic model" chip, GET STARTED CTA, and the
 * Byte the Beaver card on the right with "YOUR PERSONAL AI TUTOR" below it. */
export function HeroSection() {
  return (
    <section className="flex flex-col items-center gap-16 py-16 md:py-20 lg:flex-row lg:items-start lg:justify-between lg:gap-12 lg:py-24">
      <Reveal className="flex w-full max-w-[560px] flex-col items-start gap-8">
        <h1 className="text-gradient-display font-display text-display-1 font-bold tracking-[-4.8px] uppercase">
          <span className="block">Master</span>
          <span className="glow-text-cyan block">DSA</span>
        </h1>

        <div className="flex max-w-[512px] flex-col items-start gap-2">
          <p className="font-mono text-h2 text-tv-text-nav">
            Elevate your competitive programming with our
          </p>
          <span className="glow-card-cyan w-fit rounded-tv-chip border border-tv-border-cyan bg-tv-cyan/10 px-3 py-2 font-mono text-h2 font-bold text-tv-text-hi">
            Intelligent learning heuristic model
          </span>
          <span className="font-body text-h2 text-tv-text-nav">.</span>
        </div>

        <Button
          render={<Link href="/onboarding" />}
          variant="outline-cyan"
          className="h-auto rounded-tv-chip px-8 py-4 text-h3 text-tv-text-hi"
        >
          Get Started
        </Button>
      </Reveal>

      <Reveal delay={0.1} className="flex w-full max-w-[512px] flex-col items-center gap-6">
        <div className="relative flex w-full items-center justify-center overflow-hidden rounded-tv-card p-10 md:p-14">
          <div className="glass-panel glow-card-cyan absolute inset-0 -rotate-2 rounded-tv-card border-tv-border-cyan" />
          <div className="relative flex items-center gap-6">
            <Image
              src="/figma/byte-mascot-hero.png"
              alt="Byte the Beaver, Transverse's mascot"
              width={461}
              height={541}
              className="h-auto w-[140px] rotate-1 object-contain md:w-[180px]"
              priority
            />
            <span className="font-display max-w-[9ch] text-[36px] leading-[1.05] font-bold text-tv-text-hi uppercase blur-[1px] md:text-[44px]">
              Byte the Beaver
            </span>
          </div>
        </div>

        <p className="font-display text-h1 text-center text-tv-text-hi uppercase">
          Your Personal AI Tutor
        </p>
      </Reveal>
    </section>
  );
}
