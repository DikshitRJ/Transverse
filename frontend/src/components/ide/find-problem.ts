/**
 * Fetches one `ProblemPayload` by id for `/solve/[problemId]`.
 *
 * Backend gap worth flagging explicitly: there is no `GET /problems/{id}`
 * route. `backend/cmd/server/main.go` only mounts `GET /problems/search`
 * and `POST /problems/scrape` (verified against the Go source — see also
 * `lib/api/endpoints.ts`, which likewise only exposes `searchProblems`/
 * `scrapeProblem`, no "get by id"). plan.md §2's route table for `/solve/
 * [problemId]` only lists `POST /execute`, `GET /execute/{token}`,
 * `POST /execute/batch` as this route's backend surface — it doesn't say
 * how the problem statement itself gets loaded, which is exactly this gap.
 *
 * Until a real `GET /problems/{id}` exists, this pages through
 * `searchProblems` (the only read endpoint problems are exposed through)
 * and matches by id client-side. Against the MSW mocks (40 fixtures, well
 * under one page) this resolves in a single request. Against a live
 * backend with a large catalog this degrades to needing multiple pages —
 * bounded here to 10 pages / 1000 problems scanned so a bad id fails fast
 * instead of paging forever. Recommend a real `GET /problems/{id}` route
 * to whoever picks up the backend gap list next.
 */
import { searchProblems } from "@/lib/api/endpoints";
import type { ProblemPayload } from "@/lib/api/types";

const PAGE_SIZE = 100;
const MAX_PAGES = 10;

export async function findProblemById(problemId: string): Promise<ProblemPayload | null> {
  let offset = 0;

  for (let page = 0; page < MAX_PAGES; page += 1) {
    const { problems, total } = await searchProblems({ limit: PAGE_SIZE, offset });
    const match = problems.find((p) => p.id === problemId);
    if (match) return match;

    offset += problems.length;
    if (problems.length === 0 || offset >= total) break;
  }

  return null;
}
