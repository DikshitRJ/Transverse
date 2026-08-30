# Transverse

Transverse is an adaptive learning platform for data structures and competitive programming. It combines a Go API, a Next.js frontend, a PostgreSQL/pgvector data store, Redis, Judge0, and MinIO to guide practice, verify skills, and generate topic roadmaps.

The project can run in two useful modes:

- **Frontend mock mode** is the quickest way to explore the UI. It requires only Node.js and uses local MSW fixtures instead of the API.
- **Full Docker stack** runs the frontend, API, database, cache, code runner, object storage, and seed pipeline together.

## Prerequisites

- Node.js 22+ and npm for frontend mock mode.
- Docker Desktop with Docker Compose for the complete local stack.

## Quick start: frontend mock mode

This mode needs no database, Go installation, Docker, or API key.

```bash
cd frontend
cp .env.example .env.local
npm install
npm run dev
```

Open <http://localhost:3000>. `frontend/.env.local` defaults to `NEXT_PUBLIC_API_MODE=mock`, so MSW supplies the API responses.

## Quick start: complete local stack

Run these commands from the repository root after Docker Desktop is running:

```bash
docker compose up --build -d --scale tunnel=0
docker compose ps
docker compose logs -f init-data backend frontend
```

The first run downloads images, builds the Go and Next.js applications, and seeds the local database. `init-data` should finish with exit code `0` before the backend starts. Press `Ctrl+C` to stop following logs; the containers continue running in the background.

Open these local services once startup is complete:

| Service | Address |
| --- | --- |
| Frontend | <http://localhost:3000> |
| Backend health check | <http://localhost:8080/health> |
| MinIO console | <http://localhost:9001> |
| Judge0 API | <http://localhost:2358> |

`--scale tunnel=0` intentionally disables the Cloudflare tunnel service. Do not start that service unless you have reviewed its configuration and supplied your own tunnel credentials.

To stop the stack while retaining local database and MinIO volumes:

```bash
docker compose down
```

`docker compose down -v` additionally deletes all local database and object storage data, so use it only when you intend to reset the environment.

## Configuration

Docker Compose supplies development-safe container settings for the local stack, including a local database, Redis, MinIO, and a development auth bypass. Do not use those defaults in a deployed environment.

The LLM-assisted features are optional for startup. To enable them, export a valid Z.ai key before bringing up (or recreating) the backend:

```bash
export ZAI_API_KEY="your-key"
docker compose up -d --force-recreate backend
```

For non-Docker backend development, begin with `backend/.env.example`. For a locally run frontend that talks to a local backend, set the following in `frontend/.env.local` and restart `npm run dev`:

```dotenv
NEXT_PUBLIC_API_MODE=live
BACKEND_URL=http://localhost:8080
```

## Repository layout

```text
backend/           Go API, migrations, pipelines, and container build
frontend/          Next.js application
data/              Topic graph plus problem and tutorial seed data
Documentation/     OpenAPI contract, architecture notes, and walkthroughs
extras/            Reference material and one-off development utilities
docker-compose.yml Local service orchestration
```

## Verification

Run checks from the relevant directory:

```bash
cd backend && go test ./...
cd frontend && npm run lint && npm run typecheck && npm test
```

See [CODEBASE.md](CODEBASE.md) for the architecture map and [AGENTS.md](AGENTS.md) for contribution constraints.
