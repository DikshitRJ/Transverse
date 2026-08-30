/**
 * The 8 languages the backend actually generates templates for — mirrors
 * `backend/internal/templates/templates.go:GenerateTemplates`, whose map
 * keys are `py`, `cpp`, `java`, `js`, `go`, `rust`, `c`, `kt` (NOT
 * `typescript`/`csharp` — a real backend gotcha worth getting right in the
 * mocks). `language_id` values are the standard Judge0 CE ids.
 */
export interface LanguageMeta {
  key: string;
  label: string;
  judge0Id: number;
  monacoId: string;
}

export const LANGUAGES: LanguageMeta[] = [
  { key: "py", label: "Python 3", judge0Id: 71, monacoId: "python" },
  { key: "cpp", label: "C++17", judge0Id: 54, monacoId: "cpp" },
  { key: "java", label: "Java", judge0Id: 62, monacoId: "java" },
  { key: "js", label: "JavaScript", judge0Id: 63, monacoId: "javascript" },
  { key: "go", label: "Go", judge0Id: 60, monacoId: "go" },
  { key: "rust", label: "Rust", judge0Id: 73, monacoId: "rust" },
  { key: "c", label: "C", judge0Id: 50, monacoId: "c" },
  { key: "kt", label: "Kotlin", judge0Id: 78, monacoId: "kotlin" },
];

export function languageByJudge0Id(id: number): LanguageMeta | undefined {
  return LANGUAGES.find((l) => l.judge0Id === id);
}

function toFunctionName(slug: string): string {
  if (!slug) return "solve";
  return slug
    .split("-")
    .filter(Boolean)
    .map((p) => p[0]!.toUpperCase() + p.slice(1).toLowerCase())
    .join("");
}

/** Mirrors the shape (not the exact bytes) of `templates.GenerateTemplates`. */
export function generateTemplates(problemName: string, slug: string): Record<string, string> {
  const name = problemName || "Problem";
  const fn = toFunctionName(slug) || "solve";
  return {
    py: `# ${name}\ndef ${fn.charAt(0).toLowerCase() + fn.slice(1)}():\n    # write your solution here\n    pass\n`,
    cpp: `// ${name}\n#include <bits/stdc++.h>\nusing namespace std;\n\nvoid ${fn.charAt(0).toLowerCase() + fn.slice(1)}() {\n    // write your solution here\n}\n\nint main() {\n    ${fn.charAt(0).toLowerCase() + fn.slice(1)}();\n    return 0;\n}\n`,
    java: `// ${name}\npublic class Solution {\n    public static void ${fn.charAt(0).toLowerCase() + fn.slice(1)}() {\n        // write your solution here\n    }\n\n    public static void main(String[] args) {\n        ${fn.charAt(0).toLowerCase() + fn.slice(1)}();\n    }\n}\n`,
    js: `// ${name}\nfunction ${fn.charAt(0).toLowerCase() + fn.slice(1)}() {\n  // write your solution here\n}\n`,
    go: `// ${name}\npackage main\n\nfunc ${fn}() {\n\t// write your solution here\n}\n\nfunc main() {\n\t${fn}()\n}\n`,
    rust: `// ${name}\nfn ${fn.charAt(0).toLowerCase() + fn.slice(1)}() {\n    // write your solution here\n}\n\nfn main() {\n    ${fn.charAt(0).toLowerCase() + fn.slice(1)}();\n}\n`,
    c: `// ${name}\n#include <stdio.h>\n\nvoid ${fn.charAt(0).toLowerCase() + fn.slice(1)}(void) {\n    // write your solution here\n}\n\nint main(void) {\n    ${fn.charAt(0).toLowerCase() + fn.slice(1)}();\n    return 0;\n}\n`,
    kt: `// ${name}\nfun ${fn.charAt(0).toLowerCase() + fn.slice(1)}() {\n    // write your solution here\n}\n\nfun main() {\n    ${fn.charAt(0).toLowerCase() + fn.slice(1)}()\n}\n`,
  };
}
