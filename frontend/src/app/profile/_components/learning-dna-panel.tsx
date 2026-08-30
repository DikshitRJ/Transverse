import { formatPercent, formatTopicLabel } from "@/components/charts/chart-theme";
import type { LearningDNA } from "@/lib/api/types";

function formatDuration(ms: number): string {
  const totalSeconds = Math.round(ms / 1000);
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  if (minutes === 0) return `${seconds}s`;
  return `${minutes}m ${seconds.toString().padStart(2, "0")}s`;
}

function formatHour(hour: number): string {
  const h = ((Math.round(hour) % 24) + 24) % 24;
  const period = h < 12 ? "AM" : "PM";
  const twelve = h % 12 === 0 ? 12 : h % 12;
  return `${twelve} ${period}`;
}

function DnaStat({ label, value, caption }: { label: string; value: string; caption?: string }) {
  return (
    <div className="flex flex-col gap-0.5">
      <span className="font-mono text-[11px] tracking-wide text-tv-text-body uppercase">{label}</span>
      <span className="font-heading text-lg text-tv-text-hi">{value}</span>
      {caption && <span className="font-body text-xs text-tv-text-body">{caption}</span>}
    </div>
  );
}

/**
 * The Learning DNA object — decoded server-side (`GET /user/profile`'s
 * `UserProfileResponse.dna`) into concrete numbers, unlike the raw JSONB on
 * `GET /auth/me`. Presented as plain labeled stats rather than a chart:
 * per `dataviz`'s form guidance, "a handful of headline numbers" is a KPI
 * row, not a chart forced onto single data points.
 */
export function LearningDnaPanel({ dna }: { dna: LearningDNA }) {
  const topicBiasEntries = Object.entries(dna.topic_bias)
    .sort((a, b) => b[1] - a[1])
    .slice(0, 6);
  const maxBias = Math.max(1, ...topicBiasEntries.map(([, v]) => v));

  return (
    <div className="flex flex-col gap-6">
      <div className="grid grid-cols-2 gap-x-6 gap-y-4 sm:grid-cols-3 lg:grid-cols-4">
        <DnaStat label="Avg accuracy" value={formatPercent(dna.avg_accuracy)} />
        <DnaStat label="Avg time / problem" value={formatDuration(dna.avg_time_taken_ms)} />
        <DnaStat label="Solve velocity" value={dna.avg_solve_velocity.toFixed(2)} caption="problems / min" />
        <DnaStat label="Carelessness index" value={formatPercent(dna.carelessness_index)} caption="lower is better" />
        <DnaStat label="Peak performance" value={formatHour(dna.peak_performance_hour)} />
        <DnaStat label="Avg session length" value={`${dna.avg_session_length.toFixed(0)} min`} />
        <DnaStat label="Total sessions" value={dna.total_sessions.toLocaleString()} />
        <DnaStat label="Problems solved" value={dna.total_problems_solved.toLocaleString()} />
        <DnaStat label="Best streak" value={String(dna.streak_record)} caption="sessions" />
        <DnaStat label="Preferred platform" value={dna.preferred_platform || "—"} />
      </div>

      {topicBiasEntries.length > 0 && (
        <div>
          <p className="mb-2 font-mono text-[11px] tracking-wide text-tv-text-body uppercase">Topic focus bias</p>
          <ul className="flex flex-col gap-2">
            {topicBiasEntries.map(([topic, bias]) => (
              <li key={topic} className="flex items-center gap-3">
                <span className="w-32 shrink-0 truncate font-body text-xs text-tv-text-body">
                  {formatTopicLabel(topic)}
                </span>
                <div className="h-1.5 flex-1 overflow-hidden rounded-full bg-tv-surface-2">
                  <div
                    className="h-full rounded-full bg-tv-cyan"
                    style={{ width: `${Math.max(4, (bias / maxBias) * 100)}%` }}
                  />
                </div>
                <span className="w-10 shrink-0 text-right font-mono text-xs text-tv-text-hi tabular-nums">
                  {bias.toFixed(2)}
                </span>
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}
