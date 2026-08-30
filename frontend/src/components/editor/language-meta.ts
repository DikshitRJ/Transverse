import { LANGUAGES, type LanguageMeta } from "@/mocks/fixtures/languages";

const FALLBACK_LANGUAGE: LanguageMeta = { key: "py", label: "Python 3", judge0Id: 71, monacoId: "python" };

/** Looks up a `LanguageMeta` by its fixture key, falling back to Python 3
 * (never `undefined`) for an unrecognized key rather than requiring a
 * non-null assertion at every call site. */
export function findLanguageMeta(key: string): LanguageMeta {
  return LANGUAGES.find((l) => l.key === key) ?? FALLBACK_LANGUAGE;
}
