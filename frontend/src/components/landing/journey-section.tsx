import { Reveal } from "./reveal";
import { SvgIcon } from "./svg-icon";

type Corner = "tl" | "tr" | "bl" | "br";

const NODES: {
  key: string;
  title: string;
  description: string;
  icon: string;
  corner: Corner;
}[] = [
  {
    key: "evaluate",
    title: "Evaluate",
    description: "Evaluate users previous works and projects to assess skill level in DSA and CP.",
    icon: "/figma/icon-journey-evaluate.svg",
    corner: "tl",
  },
  {
    key: "learn",
    title: "Learn",
    description: "Learn from the curated course plan that is tailored to fill in your knowledge gaps.",
    icon: "/figma/icon-journey-learn.svg",
    corner: "tr",
  },
  {
    key: "master",
    title: "Master",
    description:
      "Master the learnt skillsets using our heuristic model which give you adaptive DSA problems.",
    icon: "/figma/icon-journey-master.svg",
    corner: "bl",
  },
  {
    key: "practice",
    title: "Practice",
    description:
      "Practice using the custom curated quiz to determine your true level with our heuristic model.",
    icon: "/figma/icon-journey-practice.svg",
    corner: "br",
  },
];

const CORNER_CLASSES: Record<Corner, string> = {
  tl: "-left-6 -top-6",
  tr: "-right-6 -top-6",
  bl: "-left-6 -bottom-6",
  br: "-right-6 -bottom-6",
};

/** Figma `61:153` heading + `61:156` 2x2 grid (Evaluate/Learn/Master/Practice)
 * with corner icon badges and the `61:206` connector arrows behind them. */
export function JourneySection() {
  return (
    <section className="flex flex-col items-center gap-16 py-16 md:py-24">
      <Reveal className="flex flex-col items-center gap-1 text-center">
        <h2 className="font-display text-[32px] leading-[1.15] font-bold tracking-[-1.5px] text-tv-text-hi uppercase md:text-[60px] md:tracking-[-3px]">
          Your Learning Journey Begins Here
        </h2>
        <p className="font-display text-[32px] leading-[1.15] font-bold tracking-[-1.5px] text-tv-text-body uppercase md:text-[60px] md:tracking-[-3px]">
          Follow The Path To Mastery
        </p>
      </Reveal>

      <div className="relative w-full max-w-[896px]">
        <SvgIcon
          src="/figma/icon-journey-connectors.svg"
          alt=""
          className="pointer-events-none absolute inset-[19%_18%_19%_16%] hidden object-contain md:block"
        />

        <div className="relative grid grid-cols-1 gap-y-20 md:grid-cols-2 md:gap-x-[256px] md:gap-y-[128px]">
          {NODES.map(({ key, title, description, icon, corner }) => (
            <Reveal key={key} delay={0.04} className="relative">
              <div className="glass-panel h-full rounded-tv-card border-tv-border-cyan px-6 py-8">
                <h3 className="font-display text-h3 font-bold text-tv-cyan-pure uppercase">
                  {title}
                </h3>
                <p className="mt-3 font-mono text-sm text-tv-text-body">{description}</p>
              </div>
              <div
                className={`glow-card-cyan absolute flex size-12 items-center justify-center rounded-tv-btn border border-tv-border-cyan bg-tv-bg-page ${CORNER_CLASSES[corner]}`}
              >
                <SvgIcon src={icon} alt="" className="size-5" />
              </div>
            </Reveal>
          ))}
        </div>
      </div>
    </section>
  );
}
