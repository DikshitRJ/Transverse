import { Reveal } from "./reveal";
import { SvgIcon } from "./svg-icon";

const PROBLEMS = [
  "Multi-Platform Switching",
  "Endless Searching For Resources",
  "Guessing Your Own Skill Gaps",
];

const SOLUTIONS: { icon: string; title: string; subtitle: string }[] = [
  {
    icon: "/figma/icon-solution-test-out.svg",
    title: "Test Out Quickly",
    subtitle: "Unified environment for immediate execution.",
  },
  {
    icon: "/figma/icon-solution-roadmap.svg",
    title: "Dynamic Roadmap",
    subtitle: "Curated, evolving path tailored to your progress.",
  },
  {
    icon: "/figma/icon-solution-diagnosis.svg",
    title: "Verified Diagnosis",
    subtitle: "Hypothesis-checked analysis of your true skill level.",
  },
];

/** Figma `61:68` — "YOUR PROBLEMS ARE OUR PROBLEMS": rose problem list on the
 * left, cyan solution list (icon + title + subtitle rows) on the right. */
export function ProblemsSolutionsSection() {
  return (
    <section className="flex flex-col items-center gap-16 py-16 md:py-20">
      <Reveal>
        <h2 className="font-display text-center text-[44px] leading-[1.05] font-bold tracking-[-2px] text-tv-text-hi uppercase md:text-[60px] md:tracking-[-3px]">
          Your Problems Are Our Problems
        </h2>
      </Reveal>

      <div className="grid w-full grid-cols-1 gap-12 md:grid-cols-2 md:gap-16">
        <Reveal className="flex flex-col items-start gap-6">
          <span className="glow-chip-rose w-fit rounded-tv-chip border border-tv-rose bg-tv-rose/10 px-4 py-3 font-display text-h1 font-bold text-tv-rose uppercase">
            Problems
          </span>
          <ul className="flex w-full flex-col gap-6">
            {PROBLEMS.map((problem) => (
              <li
                key={problem}
                className="glass-panel glow-card-rose rounded-tv-card border-transparent px-6 py-5"
              >
                <span className="font-mono text-h4 text-tv-text-hi-alt uppercase">{problem}</span>
              </li>
            ))}
          </ul>
        </Reveal>

        <Reveal delay={0.08} className="flex flex-col items-start gap-6">
          <span className="glow-card-cyan w-fit rounded-tv-chip border border-tv-cyan-pure bg-tv-cyan/10 px-4 py-3 font-display text-h1 font-bold text-tv-cyan-pure uppercase">
            Solution We Offer
          </span>
          <ul className="flex w-full flex-col gap-6">
            {SOLUTIONS.map(({ icon, title, subtitle }) => (
              <li
                key={title}
                className="glass-panel glow-card-cyan flex items-start gap-4 rounded-tv-card border-transparent px-6 py-6"
              >
                <SvgIcon src={icon} alt="" className="mt-1 size-6 shrink-0" />
                <div className="flex flex-col gap-2">
                  <span className="font-display text-h3 font-bold text-tv-cyan-pure uppercase">
                    {title}
                  </span>
                  <span className="font-mono text-sm text-tv-text-body">{subtitle}</span>
                </div>
              </li>
            ))}
          </ul>
        </Reveal>
      </div>
    </section>
  );
}
