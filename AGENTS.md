# Transverse contribution guide

Use this guide when changing the repository. It describes constraints that are important to the current implementation, not a historical ownership record.

## Architecture boundaries

- Keep HTTP work in `backend/internal/handlers`: parse and validate requests, call application services, and return HTTP responses.
- Keep business rules in `backend/internal/services`, `roadmap`, `evidence`, and other domain packages. Do not place SQL queries in handlers or services.
- Keep PostgreSQL access in `backend/internal/repository`, using the injected `pgxpool` dependency.
- Construct concrete dependencies in `backend/cmd/server/main.go`; depend on package interfaces elsewhere when a dependency needs to be substituted.
- Keep the Next.js frontend's API calls in `frontend/src/lib/api`. Do not add ad-hoc fetch clients to components.

## Determinism, LLMs, and privacy

- LLM output may help derive hypotheses, hints, error analysis, and roadmap structure. Validate structured LLM output before using it.
- Rating, scoring, correctness, and progression decisions remain deterministic application logic. Do not make an LLM the authority for those decisions.
- Do not persist raw resumes, codebase archives, or scraped profile payloads. Preserve the evidence service's cleanup behaviour when modifying intake or object-storage code.
- Never commit credentials, OAuth secrets, tunnel tokens, `.env` files, or production data.

## Async and API behaviour

- Long-running work such as external scraping, embedding, or LLM calls should use the Redis-backed jobs flow where asynchronous behaviour is already expected. Return a job identifier and publish progress through the realtime layer instead of blocking a request.
- Keep public HTTP behaviour aligned with `Documentation/openapi.yaml` and update that contract when a supported endpoint changes.
- Preserve the frontend's mock/live API split. Mock mode must remain usable without Docker or a backend.

## Local development

- Start the full local environment with `docker compose up --build -d --scale tunnel=0`. Keep the tunnel disabled unless its configuration has been explicitly reviewed for the target environment.
- Use `backend/.env.example` and `frontend/.env.example` as templates for non-container development. Never add real values to either template.
- Docker's local auth bypass and default service credentials are development conveniences only; do not copy them to production configuration.

## Tests and documentation

- Run focused tests for changed Go packages; run `go test ./...` when a backend-wide check is practical.
- For frontend changes, run the relevant combination of `npm run lint`, `npm run typecheck`, and `npm test` from `frontend/`.
- Update README, CODEBASE, OpenAPI, and schema documentation when a user-facing workflow, service, endpoint, or data shape changes.
- Keep generated data and one-off utilities under their existing `data/` and `extras/` locations; do not mix them into application packages.
