"use client";

import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import type { LanguageMeta } from "@/mocks/fixtures/languages";
import { cn } from "@/lib/utils";

export interface AnswerSurfaceProps {
  languages: LanguageMeta[];
  language: LanguageMeta;
  onLanguageChange: (language: LanguageMeta) => void;
  code: string;
  onCodeChange: (code: string) => void;
  disabled?: boolean;
  className?: string;
}

/**
 * A lightweight code-answer surface for the quiz + adaptive-practice loops.
 * Deliberately not Monaco — FOUNDATION.md §13 scopes the real editor
 * integration to FORGE's `/solve/[problemId]` IDE ("No Monaco integration
 * ... wired up yet"). A plain, monospaced textarea is enough surface area to
 * exercise the full execute -> poll -> submit handshake here.
 */
export function AnswerSurface({
  languages,
  language,
  onLanguageChange,
  code,
  onCodeChange,
  disabled,
  className,
}: AnswerSurfaceProps) {
  return (
    <div className={cn("flex flex-col gap-2", className)}>
      <div className="flex items-center justify-between gap-2">
        <Label htmlFor="answer-language" className="font-mono text-xs text-tv-text-body uppercase">
          Language
        </Label>
        <Select
          value={language.key}
          onValueChange={(key) => {
            const next = languages.find((l) => l.key === key);
            if (next) onLanguageChange(next);
          }}
          disabled={disabled}
        >
          <SelectTrigger id="answer-language" size="sm" className="w-40">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {languages.map((l) => (
              <SelectItem key={l.key} value={l.key}>
                {l.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <Textarea
        value={code}
        onChange={(e) => onCodeChange(e.target.value)}
        disabled={disabled}
        spellCheck={false}
        aria-label="Solution code"
        className="min-h-[260px] resize-y rounded-tv-btn border-tv-border bg-tv-surface-deep font-mono text-sm text-tv-text-hi focus-visible:border-tv-cyan focus-visible:ring-tv-cyan/30"
      />
    </div>
  );
}
