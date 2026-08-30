import type { LearningDNA, User } from "@/lib/api/types";

/**
 * Decide whether a just-authenticated user should land on `/onboarding`
 * (never engaged the product) or `/dashboard` (returning). There is no
 * explicit `onboarding_completed` flag anywhere in the system — verified
 * against `backend/internal/models/db_models.go` (`User`) and
 * `UserProfileResponse`, neither of which carries one — so this infers it
 * from `User.dna`.
 *
 * Why `dna` and not `created_at`/`updated_at`: `UserRepo.GetOrCreate`
 * (`backend/internal/repository/user_repo.go`) runs on *every* OAuth
 * callback via `INSERT ... ON CONFLICT (id) DO UPDATE SET updated_at =
 * NOW()`, so `updated_at` is bumped on every login, including a brand
 * new account's very first one — the two timestamps only coincide by
 * accident of request timing, not by design. `dna`, by contrast, is
 * seeded via `models.DefaultDNA()` (`total_sessions: 0,
 * total_problems_solved: 0`) and is only ever populated by real practice
 * activity — it's a direct read of "has this person actually done
 * anything," which is the real question onboarding is asking.
 *
 * `User.dna` is raw/untyped JSONB (may be `null`, `{}`, or a serialized
 * `LearningDNA` — see the comment on `User` in `lib/api/types.ts`), so
 * this parses defensively and fails safe: anything unparseable is
 * treated as "new."
 */
export function isNewUser(user: User): boolean {
  const dna = user.dna;
  if (!dna || typeof dna !== "object") return true;

  const candidate = dna as Partial<LearningDNA>;
  const sessions = typeof candidate.total_sessions === "number" ? candidate.total_sessions : 0;
  const solved =
    typeof candidate.total_problems_solved === "number" ? candidate.total_problems_solved : 0;

  return sessions === 0 && solved === 0;
}

/** Where `/auth/callback` sends a user once tokens are captured and the profile loads. */
export function postSignInDestination(user: User): "/onboarding" | "/dashboard" {
  return isNewUser(user) ? "/onboarding" : "/dashboard";
}
