import { dirname } from "path";
import { fileURLToPath } from "url";
import { FlatCompat } from "@eslint/eslintrc";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

// eslint-config-next@15.x still ships eslintrc-style ("extends") configs,
// not flat-config arrays — bridge them via FlatCompat. (The flat-array
// `eslint-config-next/core-web-vitals` import that `create-next-app`
// scaffolds by default only works with eslint-config-next@16+; this repo
// is pinned to Next 15 per plan.md's stack decision, so this is the
// correct form here, not a workaround to "fix" later.)
const compat = new FlatCompat({
  baseDirectory: __dirname,
});

const eslintConfig = [
  ...compat.extends("next/core-web-vitals", "next/typescript"),
  {
    ignores: [".next/**", "out/**", "build/**", "next-env.d.ts", "public/mockServiceWorker.js"],
  },
];

export default eslintConfig;
