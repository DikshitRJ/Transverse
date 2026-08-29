# Velocity — v1.0.0: DPP Generator & Collab Platform

## Overview

Full-stack JEE adaptive learning platform with:

- **Adaptive learning engine** — IRT 1PL Rasch theta ladder, Glicko-2 rating, rule-based question selection
- **DPP Generator** — Daily Practice Paper creation with Gemini-powered answer keys, DOCX/PDF export
- **Biometric tracking** — Client-side camera-based attention monitoring via MediaPipe FaceLandmarker
- **Collab platform** — Lecture management, DPP builder for approved collaborators
- **Knowledge graph** — Syllabus-based chapter graph with mastery visualization
- **Leaderboard & profiles** — Public user profiles, Glicko rating display, trend charts
- **Admin panel** — Collaborator management

**Tech stack:** Go 1.26 (Chi, pgx, pgvector) · Next.js 16 + React 19 + Tailwind 4 · Python 3.13 (uv) · PostgreSQL 16 + pgvector · MediaPipe FaceLandmarker

> **Reference guides:** `versions/guides/scoring_engine.md` (IRT & question selection), `versions/guides/api_reference.md` (all endpoints), `versions/guides/graph_loading.md` (syllabus graph), `versions/guides/biometrics.md` (camera tracking), `versions/guides/preprocessing.md` (data pipeline), `versions/guides/cloudinary.md` (image serving).

---

## Architecture

### Middleware stack (applied in order)

```
Logger → Recoverer → RateLimit → Auth → AdminOnly/CollabOnly
```

All `/api/v1/*` routes require `Auth` except `/health` and `/auth/*`. Admin routes add `AdminOnly`, collab routes add `CollabOnly` + `RequireCollabType`.

### Key packages

| Package | Files | Purpose |
|---------|-------|---------|
| `cmd/server/main.go` | 1 | Entrypoint, wiring, `setupRouter` (all routes), graceful shutdown, stale session cleanup |
| `internal/handlers/` | 11 files | HTTP handlers — each file maps to one domain |
| `internal/services/` | 15 files | Business logic — scoring, DPP pipeline, auth, graph |
| `internal/repository/` | 9 files | DB access layer via pgx (no raw `database/sql`) |
| `internal/models/` | 3 files | `db_models.go` (DB structs with JSONB parsers) + `dto.go` (API response types) |
| `internal/middleware/` | 4 files | Auth, AdminOnly, CollabOnly, RateLimit, collab type enforcement |
| `internal/config/` | 1 file | Env-based config with defaults |
| `internal/graph/` | 1 file | Syllabus JSON loader + prerequisite resolver |
| `internal/cache/` | 2 files | In-memory cache interface + Noop fallback |
| `internal/database/` | 1 file | pgxpool creation with ping verification |
| `internal/cloudinary/` | 1 file | Signed URL generation for authenticated image assets |

---

## All API Routes

### Auth (`/auth`)

| Method | Path | Handler | Purpose |
|--------|------|---------|---------|
| GET | `/auth/login` | `Login` | Redirect to Alpha-Auth OAuth2 with state cookie |
| POST | `/auth/exchange` | `Exchange` | Exchange auth code for JWT session token |
| POST | `/auth/validate` | `Validate` | Validate JWT token, return user info |

### Learn (`/api/v1/learn`)

| Method | Path | Handler | Purpose |
|--------|------|---------|---------|
| POST | `/api/v1/learn/start` | `StartSession` | Create session with scope (chapters/groups/subjects), first question |
| POST | `/api/v1/learn/submit` | `SubmitAnswer` | Submit answer, IRT theta update, next question |
| POST | `/api/v1/learn/skip` | `SkipQuestion` | Skip question (no theta change), next question |
| POST | `/api/v1/learn/close` | `CloseSession` | End session, Glicko-2 update, compute DNA |
| GET | `/api/v1/learn/session` | `GetSession` | Resume interrupted session |
| GET | `/api/v1/learn/chapters` | `GetChapters` | List all chapters with stats |
| GET | `/api/v1/learn/page` | `GetLearnPage` | Learn page data (chapters, last session, exam types, default settings) |
| GET | `/api/v1/learn/filters` | `GetFilters` | Available exam types and years |
| POST | `/api/v1/learn/similar` | `ShowSimilar` | Fetch similar question (same chapter, different difficulty) |
| GET | `/api/v1/learn/session/analysis` | `GetSessionAnalysis` | Full session breakdown |
| GET | `/api/v1/learn/history` | `GetSessionHistory` | Past sessions list |

### Biometric (`/api/v1/learn/biometrics`) — same router group

| Method | Path | Handler | Purpose |
|--------|------|---------|---------|
| POST | `/api/v1/learn/biometrics/sync` | `Sync` | Flush telemetry buffer |
| POST | `/api/v1/learn/biometrics/close` | `Close` | Finalize biometric session, update DNA |
| GET | `/api/v1/learn/biometrics/{sessionId}` | `Get` | Retrieve biometric data for analysis |

### Dashboard & Graph

| Method | Path | Handler | Purpose |
|--------|------|---------|---------|
| GET | `/api/v1/dashboard` | `GetDashboard` | Stats, performance trend, recent activity (range: week/month/year/all/custom) |
| GET | `/api/v1/graph` | `GetGraph` | Knowledge graph with user mastery |

### Leaderboard

| Method | Path | Handler | Purpose |
|--------|------|---------|---------|
| GET | `/api/v1/leaderboard` | `Get` | Leaderboard (mode=learn/contest/mock, search filter) |

### Settings

| Method | Path | Handler | Purpose |
|--------|------|---------|---------|
| GET | `/api/v1/settings` | `GetSettings` | User settings (notifications, beta, profile, social connections) |
| PUT | `/api/v1/settings` | `UpdateSettings` | Update settings |

### Public Profiles

| Method | Path | Handler | Purpose |
|--------|------|---------|---------|
| GET | `/api/v1/users/by-username/{username}/profile` | `GetByUsername` | Public profile with ratings |
| GET | `/api/v1/users/by-username/{username}/trend` | `GetTrend` | User performance trend |
| GET | `/api/v1/users/by-username/{username}/graph` | `GetGraph` | User knowledge graph |
| GET | `/api/v1/users/search` | `SearchUsers` | Search users by username |

### Admin (`/api/v1/admin`) — Auth + AdminOnly

| Method | Path | Handler | Purpose |
|--------|------|---------|---------|
| GET | `/api/v1/admin/collaborators` | `ListCollaborators` | List all collaborators |
| POST | `/api/v1/admin/collaborators` | `AddCollaborator` | Add collaborator |
| DELETE | `/api/v1/admin/collaborators/{email}` | `RemoveCollaborator` | Remove collaborator |

### Debug (`/api/v1/debug`) — Auth + AdminOnly

| Method | Path | Handler | Purpose |
|--------|------|---------|---------|
| GET | `/api/v1/debug/sessions` | `GetAllSessions` | All sessions across all users (with scoring details) |

### Collab (`/api/v1/collab`) — Auth + CollabOnly + RequireCollabType("dpp")

| Method | Path | Handler | Purpose |
|--------|------|---------|---------|
| GET | `/collab/me` | `MyCollabInfo` | Current collaborator info |
| POST | `/collab/lectures` | `UploadLecture` | Create lecture |
| GET | `/collab/lectures` | `ListLectures` | List lectures |
| DELETE | `/collab/lectures/{id}` | `DeleteLecture` | Delete lecture |
| POST | `/collab/dpps/preview` | `PreviewDPP` | Preview question selection for DPP |
| POST | `/collab/dpps/finalize` | `FinalizeDPP` | Create DPP from pre-selected questions |
| POST | `/collab/dpps/generate` | `GenerateDPP` | Auto-generate DPP with Gemini answer key |
| GET | `/collab/dpps` | `ListDPPs` | List DPPs |
| GET | `/collab/dpps/{id}` | `GetDPP` | DPP detail with full questions |
| DELETE | `/collab/dpps/{id}` | `DeleteDPP` | Delete DPP |
| GET | `/collab/dpps/{id}/export/docx` | `ExportDPPDOCX` | Export as DOCX |
| GET | `/collab/dpps/{id}/export/pdf` | `ExportDPPPDF` | Export as PDF |

### Health

| Method | Path | Handler | Purpose |
|--------|------|---------|---------|
| GET | `/health` | Inline | `{"status":"ok","pool_total":N}` |

---

## Database Tables

### `questions`
Core question bank. Types: `MCQ`, `MULTI_CORRECT`, `NUMERICAL`. JSONB `options` and `images`, VECTOR(384) `embedding` for cosine similarity search. Glicko-2 fields per question. HNSW index on embedding.

### `users`
Glicko-2 ratings for 3 modes (`learn_rating`, `contest_rating`, `mock_rating`). `learning_dna` JSONB (accuracy, velocity, fatigue tolerance, carelessness, subject bias, etc.). `biometric_dna` JSONB. `settings` JSONB. `social_connections` JSONB (reddit, spotify, discord, github).

### `learn_sessions`
Per-user sessions with `scope` JSONB (chapters, groups, subjects), `responses` JSONB array (per-question results with theta changes), `biometric_logs` JSONB, `biometric_baseline` JSONB, `biometric_dna_snapshot` JSONB. Status: `ACTIVE` / `COMPLETED` / `ABANDONED`.

### `learning_stats`
Per-user chapter-level stats. `chapters` JSONB keyed by chapter name → `ChapterStats` (theta, glicko_rating, mastery_score, totals, etc.).

### `user_question_stats`
Per-user per-question tracking: attempt_count, correct_count, total_time_ms, last_correct, timestamps.

### `collaborators`
Collab access control: email (PK), added_by, types[] (e.g. `{dpp}`), is_active.

### `lectures`
Collab lecture storage: title, description, pdf_url, pages, subject, chapters[], uploaded_by.

### `dpps`
DPP metadata: title, difficulty, subject, chapters[], mode (chapter/lecture), lecture_ids[], question_ids[], status (draft/published), docx_url, pdf_url, metadata JSONB.

### `dpp_questions`
Junction table: dpp_id → question_id, sort_order, answer, lecture_ref. `ON DELETE CASCADE` on question_id.

### Indexes
GIN indexes on `questions.options`, `learning_stats.chapters`, `users.settings`. HNSW index on `questions.embedding`. Partial index on active sessions.

---

## Frontend Pages

| Route | File | Purpose |
|-------|------|---------|
| `/` | `app/page.tsx` | Landing page with "Dashboard" CTA |
| `/dashboard` | `app/dashboard/page.tsx` | Stats cards, performance trend chart (Recharts), knowledge graph (D3), recent activity |
| `/learn` | `app/learn/page.tsx` | Chapter list + session player (question display, answer submission, skip, similar, biometric overlay) |
| `/auth/callback` | `app/auth/callback/page.tsx` | OAuth2 code exchange handler |
| `/settings` | `app/settings/page.tsx` | Profile, notifications, beta (camera toggle), social connections (reddit/spotify/discord/github) |
| `/history` | `app/history/page.tsx` | Past sessions list |
| `/history/session/[id]` | `app/history/session/[id]/page.tsx` | Session analysis (theta chart, per-question review, biometric summary) |
| `/leaderboard` | `app/leaderboard/page.tsx` | Leaderboard with search and mode filter |
| `/u/[username]` | `app/u/[username]/page.tsx` | Public profile with ratings, knowledge graph, performance trend, social links |
| `/collab` | `app/collab/page.tsx` | Collab dashboard (lectures, DPPs overview) |
| `/collab/lectures` | `app/collab/lectures/page.tsx` | Lecture CRUD |
| `/collab/dpps` | `app/collab/dpps/page.tsx` | DPP list |
| `/collab/dpps/generate` | `app/collab/dpps/generate/page.tsx` | DPP generation form |
| `/collab/dpps/[id]` | `app/collab/dpps/[id]/page.tsx` | DPP detail / editor |
| `/admin` | `app/admin/page.tsx` | Collaborator management (admin only, email-hardcoded) |
| `/debug` | `app/debug/page.tsx` | Session replay with full scoring details (admin only) |

### Key frontend architecture

- **API client**: `lib/api.ts` — Axios with Bearer token interceptor, 401 auto-logout
- **State**: Zustand stores — `useAuthStore` (token, userID, email, username + persist), `useUIStore` (sidebar), `useBiometricStore` (camera toggle + persist)
- **Biometrics**: `hooks/useBiometricState.ts` — MediaPipe FaceLandmarker, EAR/PERCLOS/blink computation, 30s sync interval, 10s calibration phase
- **Components**: `Sidebar.tsx`, `UserMenu.tsx`, `StartSessionModal.tsx`, `KnowledgeGraph.tsx` (D3 force-directed), `LatexText.tsx`, `BiometricBar.tsx`, `BiometricGauges.tsx`, `BiometricToggle.tsx`

---

## Engine (Python Data Pipeline)

Pipeline: `raw JSON` → `rate.py` → `generate_embeddings.py` → `seed.py` → PostgreSQL

| Script | Input | Output | Purpose |
|--------|-------|--------|---------|
| `rate.py` | `data/24_25_26_&_adv/` | `data/rated/` | Two-pass Rasch logit + time-divergence rating |
| `generate_embeddings.py` | `data/rated/` | `data/processed/` | 384-dim BGE embeddings via `BAAI/bge-small-en-v1.5` (8 workers) |
| `seed.py` | `data/processed/` | DB `questions` table | Filename parsing → MD5 IDs → upsert with embeddings |
| `search_by_text.py` | DB | CLI | Semantic search by cosine similarity |
| `test_embeddings.py` | DB | CLI | Interactive embedding quality test |
| `scripts/upload_images.py` | JSON files | Cloudinary | Download, watermark removal, Cloudinary upload |
| `scripts/remove_watermark.py` | Images | Images | ExamGOAL watermark removal (luminance alpha matting) |
| `scripts/clear_cloudinary.py` | Cloudinary | — | Wipe entire Cloudinary account (danger) |

**Data scope:** 187 JSON files across JEE Main 2024 (20 shifts), 2025 (19 shifts), 2026 (19 shifts), JEE Advanced 2024-2025 (4 papers) = 62 exam shifts total.

---

## Scoring System

### Session lifecycle
```
POST /start → ACTIVE session → loop { POST /submit or /skip } → POST /close → COMPLETED
```

### 1PL Rasch theta ladder (`theta.go`)
- Theta update: `Δθ = (correct - P(θ)) * learning_rate` where `P(θ) = 1 / (1 + e^(-(θ - difficulty)))`
- Learning rate = 0.40 (actual theta change is clamped and smoothed)
- Skip = no theta change

### Question selection (`scoring.go`)
- `PickBestQuestion`: Multi-factor scorer combining:
  - **Difficulty fit** (35%) — how close question's Glicko rating is to current theta
  - **Vector similarity** (15%) — cosine similarity to previous question (encourages topic clustering)
  - **Time match** (10%) — how close expected time (from `timespent_avg_ms`) is to user's solve velocity
  - **Novelty factor** (10%) — smaller weight for already-seen questions
  - **Immediate reinforce** (15%) — if last answer was wrong, prefer easier questions; if correct, prefer slightly harder
  - **Carelessness penalty** (10%) — if last wrong answer was faster than avg, penalize similar questions
  - **Momentum** (5%) — streak-based modifier
- Top `ringSize=20` candidates loaded from DB, scored, deterministic tiebreak by `ScCore` desc then random

### Glicko-2 (`glicko.go`)
- Applied at session close (not per-submit)
- Updates user's `learn_rating`, `learn_rd`, `learn_vol`
- Per-chapter stats also get Glicko-2 update independently
- RD refresh interval: idle > 30 days → RD = 350 (full reset)

### Learning DNA recomputation
- After each session close: recomputes aggregated stats from all sessions
- Fields: avg_accuracy, avg_time_taken_ms, avg_solve_velocity, carelessness_index, peak_performance_hour, avg_session_length, total_sessions, total_questions_solved, subject_bias

---

## Biometric Tracking System

### Client-side (browser)
- MediaPipe FaceLandmarker via `@mediapipe/tasks-vision` (CPU, single face)
- 4fps frame processing, 30s server sync interval, 10s calibration phase
- Computed metrics: EAR (Eye Aspect Ratio), PERCLOS, blink rate, MAR, head pose (pitch/roll), head movement variance, brow furrow
- Compound scores: fatigue_score (0-3), distraction_score (0-3), cognitive_load_score (0-3)
- Gaze zones: `task` / `write` / `off_task`

### Server-side
- Snapshot buffer stored in `learn_sessions.biometric_logs` JSONB
- Baseline stored in `learn_sessions.biometric_baseline` JSONB
- At session close: compute summary (avg fatigue, distraction, cognitive load, time-off-task %, writing %, face absent %, duration)
- DNA updated on `users.biometric_dna` using Welford's online algorithm

---

## DPP Generation Pipeline

```
POST /dpps/generate { title, chapters, difficulty, count, mode }
  ├─ 1. Validate chapters against syllabus graph
  ├─ 2. SelectQuestions() — difficulty-weighted random sampling (trim overshoot)
  ├─ 3. GenerateAnswers() — Vertex AI Gemini prompt → JSON answer key
  │      Falls back to correct answer on parse failure
  ├─ 4. CreateWithQuestionsTx() — single DB transaction
  └─ 5. Return ready DPP
```

### Export formats
- **DOCX**: ZIP-based OOXML with questions (options) and answer key sections
- **PDF**: gofpdf A4 portrait, Helvetica/Courier, questions + answer key with lecture references

---

## Configuration (environment variables)

| Variable | Default | Required | Purpose |
|----------|---------|----------|---------|
| `PORT` | `8080` | No | HTTP listen port |
| `DATABASE_URL` | — | Yes | PostgreSQL connection string |
| `DB_POOL_MIN_CONNS` | `4` | No | Min pool connections |
| `DB_POOL_MAX_CONNS` | `20` | No | Max pool connections |
| `SYLLABUS_GRAPH_PATH` | `internal/graph/syllabus.json` | No | Path to syllabus JSON |
| `CLIENT_ID` | — | Yes | Alpha-Auth OAuth2 client ID |
| `CLIENT_SECRET` | — | Yes | Alpha-Auth OAuth2 client secret |
| `JWT_SECRET` | — | Yes | Session token signing key |
| `APP_URL` | — | Yes | Public app URL (for redirect URIs) |
| `REDIRECT_URI` | `{APP_URL}/auth/callback` | No | OAuth2 redirect |
| `AUTH_BASE_URL` | `https://auth.alphajee.com` | No | Alpha-Auth base |
| `CLOUDINARY_CLOUD_NAME` | — | Yes | Cloudinary asset serving |
| `CLOUDINARY_API_KEY` | — | Yes | Cloudinary API key |
| `CLOUDINARY_API_SECRET` | — | Yes | Cloudinary API secret |
| `ADMIN_EMAIL` | `bhaleraoayush06@gmail.com` | No | Admin account (hardcoded in frontend too) |
| `CACHE_ENABLED` | `true` | No | Enable in-memory cache |
| `GOOGLE_PROJECT` | `""` | No | Vertex AI project (DPP Gemini) |
| `GOOGLE_LOCATION` | `us-central1` | No | Vertex AI location |
| `VERTEX_MODEL_ID` | `gemini-2.0-flash-001` | No | Gemini model for answer keys |
| `GOOGLE_API_KEY` | `""` | No | Gemini API key (alt to ADC) |

---

## SQL Migrations

| Migration | Purpose |
|-----------|---------|
| `sql-schemas/questions_init.sql` | Core tables: questions (with pgvector), indexes |
| `sql-schemas/learningmode_init.sql` | Users, learning_stats, learn_sessions, user_question_stats |
| `sql-schemas/007_biometric_tracking.sql` | Biometric columns on learn_sessions + users |
| `sql-schemas/008_user_settings.sql` | Settings JSONB on users |
| `sql-schemas/009_indexes.sql` | User search + rating indexes |
| `sql-schemas/010_dpp_generator.sql` | Collaborators, lectures, dpps, dpp_questions |
| `versions/db/001-all_schemas.sql` | Combined single-file schema for deployment |
| `versions/db/002-dpp_gen.sql` | ON DELETE CASCADE on dpp_questions.question_id |

---

## Key Design Decisions

### Question Selection
- Difficulty bucket weighting with configurable distribution presets (`mix`, `easy`, `medium`, `hard`)
- Light query methods skip 384-dim embedding column for performance
- Rounding overshoot trimmed via rand shuffle + slice

### Transaction Safety
- DPP creation uses single `CreateWithQuestionsTx` (was 3 separate calls)

### Answer Generation
- `mode: "lecture"` sends questions + lecture IDs to Gemini
- `mode: "chapter"` uses question's `correct` field directly
- JSON parse failures logged with raw Gemini output

### Image Serving
- All question images served via Cloudinary signed URLs with expiration
- Images signed per-request in handler using `cloudinary.Signer`

### Caching
- In-memory cache with 5-minute cleanup interval
- Used only by `LearnService` (chapter data, question lookups) and `DashboardService` (dashboard data)
- Noop fallback when disabled

---

## Test Coverage

| Package | Tests | Coverage |
|---------|-------|----------|
| `internal/models` | 19 | JSONB parsing (Options, Images, Settings, DNA, Responses, SeenQuestionIDs, Scope, BiometricLogs, BiometricBaseline), CorrectOptions formatting, DefaultUserSettings |
| `internal/repository` | 4 | DTO marshal round-trips, SessionWithUser embedding |
| **Total** | **24** | All pass |

No tests yet for: handlers, services, middleware, cache, graph, config.

---

## Fixes Applied in This Version

| Issue | Fix |
|-------|-----|
| DifficultyFromRating thresholds | Changed to match `converter.go`: <1350/1450/1550/1700 |
| No transaction in DPP creation | `CreateWithQuestionsTx` wraps in pgx tx |
| Silent Gemini JSON parse failure | Added `log.Printf` with raw output |
| Missing CASCADE on question_id FK | Migration `db/002-dpp_gen.sql` |
| SelectQuestions rounding overshoot | Trim after weighted selection |
| Embedding column loaded unnecessarily | Light query variants added |
| Preview endpoint double-load | Replaced with light count query |
| DPP ID used `sess_` prefix | Changed to `dpp_` prefix |
| No chapter/subject validation | Added `validateChapters` against syllabus graph |
| No request body size limit | `MaxBytesReader` set to 1MB |
| Skips counted as wrong in streaks | `computeStreaks` skips `Skipped` responses |
| No way to delete content | Added DELETE endpoints for lectures and DPPs |
| No server-side PDF | New PDF service via gofpdf |
| BiometricBaseline DRY violation | `BiometricBaselineDTO` → type alias of `BiometricBaseline` |
| Large main() function (173 lines, CCN=10) | Extracted into 8 focused functions with `appRepos`/`appServices`/`appHandlers` grouping structs |
| No Go tests anywhere | Added 24 tests across `models` and `repository` packages |

---

## Known Limitations

- **Bus factor = 1**: Single developer owns 94.2% of all files
- **Zero test files for handlers/services**: Only models + repository tested
- **Admin email hardcoded**: `ADMIN_EMAIL` env var + hardcoded `ADMIN_EMAIL` constant in `admin/page.tsx`
- **No rate limit on collab export endpoints**: DOCX/PDF generation could be expensive
- **Deploy is manual**: No Dockerfiles or CI/CD
- **Contest/mock modes**: Not implemented (data model has rating fields, but no endpoints)
- **RL agent, Challenge mode, Trust Score**: Aspirational v3.0 roadmap only (see `versions/guides/architecture.md`)
- **Frontend collab pages**: `/collab/dpps/generate` and `/collab/dpps/[id]` pages exist but have limited error handling
- **Vertex AI depends on environment**: Gemini API key or ADC must be configured for DPP answer key generation
