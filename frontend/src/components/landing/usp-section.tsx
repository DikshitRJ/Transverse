import { Reveal } from "./reveal";
import { SvgIcon } from "./svg-icon";

const USPS: { icon: string; title: string; description: string }[] = [
  {
    icon: "/figma/icon-usp-gamified-unlock.svg",
    title: "Gamified Progressive Unlock",
    description:
      "Experience learning as a game. Unlock new algorithms and data structures as you level up your skills, keeping motivation high.",
  },
  {
    icon: "/figma/icon-usp-learn-solve.svg",
    title: "Unified Learn-and-Solve",
    description:
      "Don't just read. Write code immediately in our integrated IDE environment tied directly to the learning materials.",
  },
  {
    icon: "/figma/icon-usp-closed-loop.svg",
    title: "Closed-Loop Mastery",
    description:
      "Our system identifies your weaknesses and loops you back to targeted practice until mastery is definitively achieved.",
  },
];

/** Figma `61:151` heading + `61:143`/`60:19`/`60:28` — the three USP cards:
 * gamified unlock, unified learn-and-solve, closed-loop mastery. */
export function UspSection() {
  return (
    <section className="flex flex-col items-center gap-16 py-16 md:py-20">
      <Reveal>
        <h2 className="font-display text-center text-[44px] leading-[1.05] font-bold tracking-[-2px] text-tv-text-hi uppercase md:text-[60px] md:tracking-[-3px]">
          We Take A Unique Approach
        </h2>
      </Reveal>

      <div className="grid w-full grid-cols-1 gap-6 md:grid-cols-3">
        {USPS.map(({ icon, title, description }, index) => (
          <Reveal key={title} delay={index * 0.06}>
            <div className="flex h-full flex-col items-start gap-4 rounded-tv-card border border-tv-border-muted bg-tv-surface/60 p-8 backdrop-blur-[6px]">
              <div className="flex size-12 items-center justify-center rounded-tv-btn border border-tv-border-muted bg-tv-bg-page">
                <SvgIcon src={icon} alt="" className="size-6" />
              </div>
              <h3 className="font-display text-h2 font-bold text-tv-text-hi uppercase">{title}</h3>
              <p className="font-body text-body text-tv-text-body">{description}</p>
            </div>
          </Reveal>
        ))}
      </div>
    </section>
  );
}
