"use client";

/**
 * Built off `LANGUAGES` (`src/mocks/fixtures/languages.ts`) per the
 * FOUNDATION.md instruction to FORGE — that array mirrors the real
 * backend's 8 Judge0-backed languages (`templates.GenerateTemplates`)
 * exactly, so it's the source of truth for language keys/labels/Judge0
 * ids/Monaco ids here rather than a hand-rolled list. `src/mocks/**` is
 * outside FORGE's owned paths to *modify*, but importing this fixture data
 * read-only is exactly what FOUNDATION.md asks for.
 */
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { LANGUAGES } from "@/mocks/fixtures/languages";

export interface LanguageSwitcherProps {
  value: string;
  onChange: (languageKey: string) => void;
  disabled?: boolean;
}

export function LanguageSwitcher({ value, onChange, disabled }: LanguageSwitcherProps) {
  return (
    <Select
      value={value}
      onValueChange={(next) => {
        if (typeof next === "string") onChange(next);
      }}
      disabled={disabled}
    >
      <SelectTrigger
        size="sm"
        className="min-w-32 border-tv-border bg-tv-surface font-mono text-xs text-tv-text-hi"
        aria-label="Language"
      >
        <SelectValue placeholder="Language" />
      </SelectTrigger>
      <SelectContent className="border-tv-border bg-tv-surface font-mono text-xs">
        {LANGUAGES.map((lang) => (
          <SelectItem key={lang.key} value={lang.key} className="text-tv-text-hi">
            {lang.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
