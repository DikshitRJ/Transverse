import type { Metadata } from "next";
import Image from "next/image";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { MascotBubble } from "@/components/onboarding/mascot-bubble";

export const metadata: Metadata = {
  title: "Start your journey — Transverse",
};

/**
 * Pixel-faithful to Figma `5FBPyLzXWSFSv9ahnzTXFs` node `15:10` (plan.md
 * route #4). Pulled via `get_design_context` — see node ids in comments
 * below where a measurement isn't obvious from the class alone. The 528px
 * card height and the 69px display heading are the Figma-authored desktop
 * (1280px) values; both scale down below `lg` so nothing overflows or
 * scrolls horizontally at 1024/768/390 per the responsive requirement —
 * Figma only designed the 1280px frame, so anything below `lg` is my own
 * reasonable reduction of the same proportions, not a second Figma source.
 */
export default function OnboardingChooserPage() {
  return (
    <main className="flex flex-1 flex-col items-center justify-center px-6 py-12 md:px-16 md:py-16">
      <div className="flex w-full max-w-[896px] flex-col items-center">
        {/* 15:26 Margin — heading + subcopy */}
        <div className="flex flex-col items-center pb-10 md:pb-12">
          <h1
            className="text-gradient-display glow-text-cyan text-center font-display text-[34px] leading-[34px] font-bold tracking-[-1.5px] uppercase md:text-[52px] md:leading-[52px] md:tracking-[-3px] lg:text-[62px] lg:leading-[60px] lg:tracking-[-4px] xl:text-display-1 xl:tracking-[-4.8px]"
          >
            Start your
            <br />
            journey today
          </h1>

          <div className="mt-4 flex max-w-[576px] flex-col items-center px-4">
            <p className="text-center font-mono text-base font-bold text-white md:text-h3">
              {"Let's calibrate your learning path. How would you like to begin your Data Structures and Algorithms adventure?"}
            </p>
          </div>
        </div>

        {/* 15:31 Container — the two chooser cards */}
        <div className="flex w-full flex-col items-stretch justify-center gap-6 lg:flex-row">
          {/* 15:32 Option 1: Sync past experiences */}
          <div className="relative flex flex-1 flex-col items-center overflow-hidden rounded-tv-card border border-tv-border bg-tv-surface p-8 lg:h-[528px] lg:p-[33px]">
            <Image
              src="/figma/icon-upload-cloud.svg"
              alt=""
              width={149}
              height={123}
              className="pointer-events-none absolute top-0 right-0"
            />

            <span className="relative flex size-48 shrink-0 items-center justify-center overflow-hidden rounded-tv-pill border border-tv-border-muted bg-tv-surface-2 p-px">
              <Image
                src="/figma/github-mark.png"
                alt=""
                width={192}
                height={192}
                className="size-full rounded-tv-pill object-cover"
              />
            </span>

            <h2 className="mt-6 text-center font-display text-h2 font-bold text-tv-text-hi uppercase">
              Sync past experiences
            </h2>

            <p className="mt-4 max-w-[280px] text-center font-body text-body text-tv-text-body">
              Upload your previous project data or connect your GitHub
              repository. We&apos;ll analyze your code to customize your
              starting level.
            </p>

            <Button
              render={<Link href="/onboarding/sync" />}
              variant="ghost"
              size="lg"
              className="mt-auto w-full max-w-[200px] justify-center gap-2 rounded-tv-btn bg-tv-cyan normal-case tracking-normal text-tv-cyan-ink hover:bg-tv-cyan/90 hover:text-tv-cyan-ink"
            >
              <Image src="/figma/icon-sync-repo.svg" alt="" width={12} height={12} />
              Connect Repo
            </Button>
          </div>

          {/* 15:48 Option 2: Take a quick quiz */}
          <div className="relative flex flex-1 flex-col items-center overflow-hidden rounded-tv-card border border-tv-border bg-tv-surface p-8 lg:h-[528px] lg:p-[33px]">
            <Image
              src="/figma/icon-quiz-monitor.svg"
              alt=""
              width={139}
              height={145}
              className="pointer-events-none absolute top-0 right-0"
            />

            <span className="relative flex size-48 shrink-0 items-center justify-center overflow-hidden rounded-tv-pill">
              <Image
                src="/figma/quiz-glyph.png"
                alt=""
                width={192}
                height={192}
                className="size-full object-contain"
              />
            </span>

            <h2 className="mt-6 text-center font-display text-h2 font-bold text-tv-text-hi uppercase">
              Take a quick quiz
            </h2>

            <p className="mt-4 max-w-[280px] text-center font-body text-body text-tv-text-body">
              Answer a few rapid-fire questions to assess your current DSA
              knowledge. Perfect if you&apos;re starting fresh or want a
              quick gauge.
            </p>

            <Button
              render={<Link href="/onboarding/quiz" />}
              variant="outline-cyan"
              size="lg"
              className="mt-auto w-full max-w-[200px] justify-center gap-2 normal-case tracking-normal"
            >
              <Image src="/figma/icon-lightning-quiz.svg" alt="" width={12} height={15} />
              Start Quiz
            </Button>
          </div>
        </div>

        {/* 15:63 Mascot Mentor:margin */}
        <MascotBubble
          quote={'"Not sure? The quiz is a fun way to warm up!"'}
          className="mt-10 md:mt-12"
        />

        <div className="mt-8 text-center">
          <Link
            href="/dashboard"
            className="text-sm font-mono text-tv-text-muted hover:text-tv-cyan transition-colors underline underline-offset-4"
          >
            Skip calibration and jump straight to Dashboard →
          </Link>
        </div>
      </div>
    </main>
  );
}
