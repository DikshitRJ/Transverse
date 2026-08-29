# Transverse — DSA/CP Adaptive Heuristic Engine: Implementation Plan

> **Status:** Pending approval  
> **Target directory:** `backend/` (inside the repo)  
> **Language:** Go (100%) — zero Python, zero external scripts  
> **Database:** PostgreSQL 16 + pgvector extension  
> **Embedding model:** `BAAI/bge-small-en-v1.5` (384-dim) served via Go-native ONNX Runtime  

---

## 0. Context & Vision

**Transverse** is a "Duolingo for DSA/CP" — a single, gamified platform that eliminates platform-hopping by combining:

- **LLM-curated personalized roadmaps** for high-level structure
- **A deterministic Glicko-rated heuristic engine** for mathematically precise question selection
- **384-dimensional vector embeddings** for concept-DNA similarity mapping

This plan covers the **heuristic engine backend** — the core scoring intelligence that sits at the heart of Transverse, adapted from the reference `heuristic_model/` (built for JEE exam prep) and re-engineered for **DSA & Competitive Programming** question banks scraped from LeetCode, Codeforces, AtCoder, and CSES.

The `backend/` folder will ultimately become a **single Go binary**, containerized via Docker, serving all API endpoints for the full Transverse application. The heuristic engine is the first vertical to build — all other services (auth, roadmap, LLM integration, compiler) will be added to the same binary later.

---

## 1. Data Sources

Scraped data available at `data/generated/`:

| File | Source | Volume | Key Fields |
|------|---------|---------|-----------|
| `codeforces.json` | Codeforces | ~13,000 problems | `id`, `name`, `url`, `difficulty_rating` (800–3500), `tags[]`, `contest_id` |
| `atcoder.json` | AtCoder | ~5,000 problems | `id`, `name`, `url`, `difficulty_rating` (null for many), `tags[]` |
| `cses.json` | CSES Problem Set | ~300 problems | `id`, `name`, `url`, `difficulty_rating`, `tags[]` |
| `leetcode_index.json` | LeetCode (tagged) | ~250 problems | `id`, `name`, `url`, `tags[]` (no numeric rating) |
| `leetcode_index_seed.json` | LeetCode (NeetCode curated) | ~65 problems | `name`, `slug`, `tags[]` |
| `all_problems.json` | Merged (all above) | ~18,000+ problems | Union of all above |

---

## 2. DSA/CP Knowledge Graph (Topic DAG)

Unlike JEE's physics/chemistry/maths taxonomy, DSA/CP uses a prerequisite DAG stored as `data/topics.json`:

```
Root
├── Foundations
│   ├── Arrays & Hashing
│   ├── Two Pointers
│   ├── Sliding Window
│   └── Prefix / Suffix
├── Data Structures
│   ├── Stack (Monotonic Stack)
│   ├── Queue
│   ├── Linked List
│   ├── Trees (BST, Segment Tree, Fenwick Tree)
│   ├── Heap / Priority Queue
│   └── Trie
├── Search & Sort
│   ├── Binary Search
│   ├── Sorting
│   └── Divide & Conquer
├── Graphs
│   ├── BFS / DFS
│   ├── Topological Sort
│   ├── Union-Find (DSU)
│   ├── Shortest Path (Dijkstra, BF, Floyd-Warshall)
│   ├── Minimum Spanning Tree
│   └── Network Flow
├── Dynamic Programming
│   ├── 1-D DP
│   ├── 2-D DP / Grid DP
│   ├── Interval DP
│   ├── Tree DP
│   ├── Bitmask DP
│   └── Digit DP
├── Math & Number Theory
│   ├── Modular Arithmetic
│   ├── Combinatorics
│   ├── Sieve / Primes
│   └── Geometry
├── Strings
│   ├── String Hashing
│   ├── KMP / Z-Algorithm
│   └── Suffix Array
├── Greedy
├── Backtracking
└── Bit Manipulation
```

---

## 3. Difficulty Scale (DSA/CP adapted)

The reference model uses a Glicko scale `800–3500`. This maps directly to CP ratings:

| Glicko Range | CF Equivalent | Level |
|---|---|---|
| 800–1000 | 800–900 (Div 4 A) | Absolute beginner |
| 1000–1200 | 900–1200 (Div 3 A/B) | Foundations known |
| 1200–1400 | 1200–1400 (Div 2 A/B) | Intermediate |
| 1400–1600 | 1400–1700 (Div 2 B/C) | Upper intermediate |
| 1600–1900 | 1700–2000 (Div 2 D) | Advanced |
| 1900–2200 | 2000–2400 (Div 1 A/B) | Expert |
| 2200–2800 | 2400–3500 (Div 1 C/D/E) | Competitive |

**Seeding strategy:**
- **Codeforces:** `glicko_rating = cf_difficulty_rating` (direct, same scale)
- **LeetCode Easy:** 900 · **Medium:** 1400 · **Hard:** 1800
- **AtCoder:** Normalize `atcoder_rating` (0–4000 → 800–2800 linearly)
- **CSES:** Use provided rating if available, else 1400 (curated intermediate set)
- **User cold-start:** `θ = 1300`

---

## 4. The 7-Factor Heuristic Scorer

The reference model has a 6-factor scorer for JEE. We adapt it with **7 factors** for DSA/CP:

| # | Factor | Default Weight | DSA/CP Adaptation |
|---|--------|---|---|
| 1 | **DifficultyFit** | 35% | `1 − |thetaEff − glickoRating| / 1500` — identical math |
| 2 | **ConceptSimilarity** | 15% | Cosine similarity of 384-dim embeddings — "conceptual DNA" mapping |
| 3 | **TopicProgression** | 10% | Bonus for problems in active roadmap topic / next prerequisite node |
| 4 | **NoveltyFactor** | 10% | `1 − min(attempts/5, 0.5)` — penalizes repeated problems |
| 5 | **ImmediateReinforce** | 15% | After wrong: weight toward concept-similar, slightly easier problems |
| 6 | **PlatformDiversity** | 5% | Mild bonus for showing problems from different platforms |
| 7 | **CarelessnessPenalty** | 10% | `carelessnessIndex × (1 − difficultyFit)` — subtracted from total |

**Context-tuned weight sets:**

```
Cold Start:    DF=70%, NoF=20%, TP=10%,  rest=0%
After Correct: DF=50%, CS=15%, TP=10%,  NoF=10%, IR=5%,  PD=5%,  CP=5%
After Wrong:   DF=15%, CS=25%, TP=5%,   NoF=5%,  IR=35%, PD=5%,  CP=10%
```

---

## 5. Backend Directory Structure

```
backend/
├── cmd/
│   └── server/
│       └── main.go                    # Entrypoint, wiring, graceful shutdown
│
├── internal/
│   ├── config/
│   │   └── config.go                  # Env-based config
│   ├── database/
│   │   └── postgres.go                # pgxpool setup
│   ├── cache/
│   │   ├── cache.go                   # Cache interface
│   │   └── memory.go                  # In-memory LRU cache (TTL-based)
│   ├── embedding/
│   │   ├── provider.go                # Embedding provider interface
│   │   ├── onnx.go                    # ONNX Runtime Go bindings (bge-small-en-v1.5)
│   │   └── tokenizer.go               # BPE tokenizer (pure Go)
│   ├── graph/
│   │   ├── graph.go                   # Topic DAG interface
│   │   └── loader.go                  # Load topics.json → in-memory DAG
│   ├── models/
│   │   ├── db_models.go               # DB structs (Problem, UserProfile, Session, DNA)
│   │   └── dto.go                     # HTTP request/response types
│   ├── repository/
│   │   ├── problem_repo.go            # Problem CRUD + pgvector ANN search
│   │   ├── user_repo.go               # User CRUD
│   │   ├── session_repo.go            # Session CRUD + response append tx
│   │   ├── stats_repo.go              # topic_stats CRUD
│   │   └── problem_stats_repo.go      # per-user per-problem attempt tracking
│   ├── services/
│   │   ├── scoring.go                 # 7-factor scorer: PickBestProblem, ScoreCandidate
│   │   ├── theta.go                   # IRT 1PL Rasch theta ladder
│   │   ├── glicko.go                  # Glicko-2 session-level rating update
│   │   ├── practice_session.go        # Session state machine (Start/Submit/Skip/Close)
│   │   ├── practice_analytics.go      # Mastery, streaks, DNA recomputation
│   │   ├── practice_cache.go          # Cache helpers for sessions
│   │   ├── graph_service.go           # Topic DAG scope resolution
│   │   └── helpers.go                 # clamp, cosine, generateID
│   ├── handlers/
│   │   ├── practice_handler.go        # /api/v1/practice/* HTTP handlers
│   │   └── health_handler.go
│   └── middleware/
│       ├── auth.go                    # JWT stub (placeholder for future auth)
│       └── ratelimit.go
│
├── pipeline/
│   └── seed/
│       ├── main.go                    # CLI: go run ./pipeline/seed
│       ├── loader.go                  # Load + deduplicate all JSON sources
│       ├── normalizer.go              # Platform difficulty normalization
│       ├── embedder.go                # Batch ONNX inference (4 workers)
│       └── seeder.go                  # pgvector upsert
│
├── sql/
│   ├── 001_init_problems.sql          # problems table with pgvector(384)
│   ├── 002_init_users.sql             # users table + dna JSONB
│   ├── 003_init_sessions.sql          # practice_sessions, responses JSONB
│   ├── 004_init_stats.sql             # user_problem_stats, topic_stats
│   └── 005_indexes.sql                # HNSW index, rating indexes, GIN indexes
│
├── data/
│   └── topics.json                    # DSA/CP knowledge graph (topic DAG)
│
├── models/                            # ONNX model files (not committed to git)
│   └── bge-small-en-v1.5.onnx
│
├── go.mod
├── go.sum
├── Dockerfile                         # Multi-stage, ONNX Runtime bundled
├── docker-compose.yml                 # PostgreSQL + pgvector + backend
└── .env.example
```

---

## 6. Database Schema

### `problems`
```sql
CREATE TABLE problems (
    id                TEXT PRIMARY KEY,           -- "cf-1-A", "leetcode-two-sum"
    source            TEXT NOT NULL,              -- "codeforces"|"leetcode"|"atcoder"|"cses"
    name              TEXT NOT NULL,
    url               TEXT NOT NULL,
    slug              TEXT,
    contest_id        TEXT,
    tags              TEXT[] DEFAULT '{}',        -- raw tags from scrape
    topic             TEXT,                       -- normalized primary topic
    subtopic          TEXT,                       -- normalized subtopic
    difficulty_label  TEXT,                       -- "easy"|"medium"|"hard"|"expert"
    glicko_rating     FLOAT NOT NULL DEFAULT 1500,
    glicko_rd         FLOAT NOT NULL DEFAULT 350,
    glicko_volatility FLOAT NOT NULL DEFAULT 0.06,
    attempt_count     INT   NOT NULL DEFAULT 0,
    solve_rate        FLOAT,
    avg_time_ms       INT   DEFAULT 0,
    embedding         VECTOR(384),               -- bge-small-en-v1.5
    embed_text        TEXT,                       -- text that was embedded
    created_at        TIMESTAMPTZ DEFAULT NOW(),
    updated_at        TIMESTAMPTZ DEFAULT NOW()
);
-- HNSW index for ANN cosine search (the concept-similarity engine)
CREATE INDEX ON problems USING hnsw (embedding vector_cosine_ops);
CREATE INDEX ON problems (topic, glicko_rating);
CREATE INDEX ON problems (source, glicko_rating);
```

### `users`
```sql
CREATE TABLE users (
    id              TEXT PRIMARY KEY,
    username        TEXT UNIQUE NOT NULL,
    email           TEXT UNIQUE,
    theta           FLOAT NOT NULL DEFAULT 1300,   -- IRT ability estimate
    glicko_rating   FLOAT NOT NULL DEFAULT 1200,
    glicko_rd       FLOAT NOT NULL DEFAULT 350,
    glicko_vol      FLOAT NOT NULL DEFAULT 0.06,
    dna             JSONB DEFAULT '{}',             -- LearningDNA
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);
```

### `practice_sessions`
```sql
CREATE TABLE practice_sessions (
    id                  TEXT PRIMARY KEY,
    user_id             TEXT REFERENCES users(id),
    mode                TEXT NOT NULL DEFAULT 'ADAPTIVE', -- ADAPTIVE|REGULAR
    scope               JSONB NOT NULL,  -- {topics[], subtopics[], sources[], difficulty_range}
    theta_start         FLOAT NOT NULL,
    theta_current       FLOAT NOT NULL,
    current_problem_id  TEXT REFERENCES problems(id),
    responses           JSONB DEFAULT '[]',  -- []SessionResponse
    question_count      INT NOT NULL DEFAULT 0,
    status              TEXT NOT NULL DEFAULT 'ACTIVE', -- ACTIVE|COMPLETED|ABANDONED
    created_at          TIMESTAMPTZ DEFAULT NOW(),
    updated_at          TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX ON practice_sessions (user_id, status);
```

### `user_problem_stats`
```sql
CREATE TABLE user_problem_stats (
    user_id         TEXT REFERENCES users(id),
    problem_id      TEXT REFERENCES problems(id),
    attempt_count   INT NOT NULL DEFAULT 0,
    correct_count   INT NOT NULL DEFAULT 0,
    total_time_ms   BIGINT NOT NULL DEFAULT 0,
    last_attempted  TIMESTAMPTZ,
    PRIMARY KEY (user_id, problem_id)
);
```

### `topic_stats`
```sql
CREATE TABLE topic_stats (
    user_id         TEXT REFERENCES users(id),
    topic           TEXT NOT NULL,
    theta           FLOAT NOT NULL DEFAULT 1300,
    mastery_score   FLOAT NOT NULL DEFAULT 0,   -- 0–100
    glicko_rating   FLOAT NOT NULL DEFAULT 1200,
    attempt_count   INT NOT NULL DEFAULT 0,
    correct_count   INT NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, topic)
);
```

---

## 7. Embedding Pipeline (`pipeline/seed/`)

This replaces the Python `generate_embeddings.py` + `seed.py` entirely with Go.

### Step 1: Load
- Read all JSON files from `data/generated/`
- Merge into `[]RawProblem`, deduplicate by `id`

### Step 2: Normalize
- **Codeforces:** `glicko_rating = cf_difficulty_rating` (direct, 800–3500)
- **AtCoder:** Linear normalize `atcoder_rating` (0–4000 → 800–2800)
- **LeetCode:** Easy→900, Medium→1400, Hard→1800 from difficulty label in name/tags
- **CSES:** Use provided rating, fallback 1400
- Map raw `tags[]` → `topic` + `subtopic` via lookup table
- Build `embed_text`: `"[PROBLEM] {name} [TOPIC] {topic} [SUBTOPIC] {subtopic} [TAGS] {tags joined} [SOURCE] {source}"`

### Step 3: Embed (ONNX in Go)
- Load `bge-small-en-v1.5.onnx` via `github.com/yalue/onnxruntime_go`
- BPE tokenization (Hugging Face tokenizer JSON, pure Go)
- Batch size: 32 problems per batch
- **4 concurrent workers** (goroutines) for parallel ONNX inference
- L2-normalize each 384-dim vector (for cosine similarity via inner product)

### Step 4: Seed
- `INSERT INTO problems (..., embedding) VALUES (..., $N::vector) ON CONFLICT (id) DO UPDATE SET ...`
- Batch upsert: 100 rows per transaction
- Progress: `✓ 13,000/18,000 problems seeded`

**CLI:**
```bash
go run ./pipeline/seed \
  --data-dir ./data/generated \
  --model ./models/bge-small-en-v1.5.onnx \
  --db "postgres://transverse:transverse@localhost:5432/transverse" \
  --batch-size 32 \
  --migrate
```

---

## 8. IRT Theta Ladder (`internal/services/theta.go`)

Direct port of reference `theta.go` — no changes needed, Glicko scale already aligns:

```
θ_irt  = (θ − 1500) / 100           # centre on 1500
b_irt  = (glicko − 1500) / 100      # centre on 1500
P(θ)   = 1 / (1 + e^(-a·(θ_irt − b_irt)))
θ_new  = θ + K · timeFactor · (actual − P(θ))

a           = 1/27     (discrimination: 27 Glicko pts ≈ 1% P(correct) shift)
K           = 30       (learning rate: max theta shift per problem)
timeFactor  = clamp(expectedTimeMs / actualTimeMs, 0.3, 2.0)
θ           clamped to [800, 3500]
```

---

## 9. Effective Theta (DSA/CP modifiers)

```
thetaEff = θ_current
         + TopicBias × 200        # per-topic strength from DNA (±100 max)
         + Momentum               # +15/correct, -20/wrong streak (±60 max)
         + SessionPhase           # warm-up (<20%) = -30, cool-down (>70%) = -20
```

**TopicBias:** `DNA.TopicBias[topic]` — updated with EMA α=0.15 on session close. Same formula as JEE SubjectBias.

---

## 10. Full Session Lifecycle

```
POST /api/v1/practice/start
  1. Resolve scope → problem pool
  2. Resume existing ACTIVE session if found
  3. Load topic theta (topic_stats, fallback 1300)
  4. Filter unseen problems
  5. Cold-start: PickBestProblem(unseen, state, nil, false)
  6. Insert practice_sessions row (status=ACTIVE)
  → Returns: {session_id, first_problem, theta_start}

POST /api/v1/practice/submit
  1. Validate session + ownership
  2. Load problem (cache 24h)
  3. IRT theta update
  4. Compute streaks (scan responses backwards)
  5. Load DNA (cached 60s)
  6. Build ScState
  7. If wrong + embedding available: pgvector ANN search
     SELECT * FROM problems WHERE topic=$1 ORDER BY embedding <=> $2 LIMIT 30
  8. PickBestProblem(candidates, state, currentProblem, isCorrect)
  9. BEGIN TX: upsert user_problem_stats + append response + update theta
  10. Invalidate seen:{userID} cache
  → Returns: {is_correct, next_problem, theta_before, theta_after}

POST /api/v1/practice/close
  1. Compute: accuracy, avg_time, per-topic breakdown
  2. Glicko-2 update (session as single game vs avg problem rating)
  3. masteryScore = (θ − 1300) / (2800 − 1300) × 100
  4. Update topic_stats (theta, mastery, Glicko)
  5. Recompute LearningDNA (EMA updates: accuracy, time, velocity, topic bias)
  6. BEGIN TX: update users + topic_stats + session.status=COMPLETED
  → Returns: {mastery_score, theta_final, accuracy, session_summary}
```

---

## 11. LearningDNA (JSONB on `users.dna`)

```go
type LearningDNA struct {
    AvgAccuracy         float64            // EMA of correct/total
    AvgTimeTakenMs      int64              // mean ms per problem
    AvgSolveVelocity    float64            // problems/hour
    CarelessnessIndex   float64            // wrong rate on problems ≤ theta-200
    PeakPerformanceHour int                // 0–23 (best accuracy hour)
    AvgSessionLength    float64            // mean problems per completed session
    TotalSessions       int
    TotalProblemsSolved int
    TopicBias           map[string]float64 // topic → accuracy delta vs avg (EMA α=0.15)
    PreferredPlatform   string             // most-solved source
    StreakRecord        int                // longest consecutive correct streak
}
```

---

## 12. API Routes

```
GET  /health

# Practice (heuristic engine)
POST /api/v1/practice/start           → start adaptive session
POST /api/v1/practice/submit          → submit answer, get next problem
POST /api/v1/practice/skip            → skip problem, get next
POST /api/v1/practice/close           → end session, update ratings + DNA
GET  /api/v1/practice/session/{id}    → resume interrupted session
GET  /api/v1/practice/similar         → ANN search: similar problems
     ?problem_id=X&topic=Y&limit=5
GET  /api/v1/practice/topics          → all topics with user mastery scores

# User
GET  /api/v1/user/profile             → theta, Glicko, DNA, streak record
GET  /api/v1/user/history             → past sessions list

# Problems
GET  /api/v1/problems/search          → full-text + tag + difficulty filter
     ?q=two+pointers&topic=arrays&source=codeforces&min_rating=1000&max_rating=1600
```

---

## 13. Caching Strategy

| Key | TTL | Invalidated By | Purpose |
|-----|-----|---|---|
| `problem:{id}` | 24h | Never | Full problem with embedding |
| `seen:{userID}` | 5min | Every Submit | Attempted problem IDs + attempt counts |
| `dna:{userID}` | 60s | Session Close | LearningDNA |
| `topic_stats:{userID}` | 60s | Session Close | Per-topic theta/mastery |
| `problems:topic:{topic}` | 10min | Never | All problems for a topic |

---

## 14. Go Module Dependencies

```go
github.com/jackc/pgx/v5             // PostgreSQL driver + pgxpool
github.com/pgvector/pgvector-go     // pgvector Vector type
github.com/go-chi/chi/v5            // HTTP router
github.com/yalue/onnxruntime_go     // ONNX Runtime Go bindings
github.com/golang-jwt/jwt/v5        // JWT (auth stub)
```

The `onnxruntime_go` binding requires `libonnxruntime.so` — bundled in the Docker image. **No Python runtime.**

---

## 15. Docker Setup

```yaml
# docker-compose.yml
services:
  db:
    image: pgvector/pgvector:pg16
    environment:
      POSTGRES_DB: transverse
      POSTGRES_USER: transverse
      POSTGRES_PASSWORD: transverse
    volumes:
      - pgdata:/var/lib/postgresql/data
    ports: ["5432:5432"]

  backend:
    build: ./backend
    depends_on: [db]
    environment:
      DATABASE_URL: postgres://transverse:transverse@db:5432/transverse
      PORT: 8080
      ONNX_MODEL_PATH: /app/models/bge-small-en-v1.5.onnx
      TOPICS_GRAPH_PATH: /app/data/topics.json
    ports: ["8080:8080"]
    volumes:
      - ./backend/models:/app/models:ro
      - ./backend/data:/app/data:ro

volumes:
  pgdata:
```

```dockerfile
# backend/Dockerfile (multi-stage)
FROM golang:1.23-bullseye AS builder
WORKDIR /app
# Install ONNX Runtime shared library
RUN wget -q https://github.com/microsoft/onnxruntime/releases/download/v1.18.1/onnxruntime-linux-x64-1.18.1.tgz \
    && tar -xzf onnxruntime-linux-x64-*.tgz \
    && cp onnxruntime-linux-x64-*/lib/libonnxruntime.so* /usr/local/lib/ \
    && ldconfig
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o transverse ./cmd/server

FROM debian:bullseye-slim
COPY --from=builder /usr/local/lib/libonnxruntime.so* /usr/local/lib/
COPY --from=builder /app/transverse /app/transverse
COPY data/topics.json /app/data/topics.json
RUN ldconfig
EXPOSE 8080
CMD ["/app/transverse"]
```

---

## 16. Implementation Phases

### Phase 1 — Foundations
- `go.mod` + project skeleton
- `internal/config/` — env config
- `internal/database/` — pgxpool setup
- `sql/` — all SQL migrations + `005_indexes.sql` with HNSW
- `internal/models/db_models.go` — Problem, User, Session, DNA structs
- `internal/models/dto.go` — API types
- `internal/cache/` — in-memory cache with TTL
- `data/topics.json` — DSA/CP knowledge graph (tag→topic lookup table included)

### Phase 2 — Embedding Pipeline
- `internal/embedding/onnx.go` — ONNX Runtime wrapper (load model, batch inference)
- `internal/embedding/tokenizer.go` — BPE tokenizer (Hugging Face JSON format)
- `pipeline/seed/loader.go` — load + deduplicate all JSON sources
- `pipeline/seed/normalizer.go` — difficulty normalization per platform
- `pipeline/seed/embedder.go` — batch embedding with 4 goroutine workers
- `pipeline/seed/seeder.go` — pgvector upsert (100 rows/tx)
- `pipeline/seed/main.go` — CLI entrypoint

### Phase 3 — Heuristic Engine Core
- `internal/services/helpers.go` — clamp, cosineSimilarity, generateID
- `internal/services/theta.go` — IRT 1PL Rasch theta ladder
- `internal/services/glicko.go` — Glicko-2 session update (Illinois algorithm)
- `internal/services/scoring.go` — 7-factor WeightSet + PickBestProblem + ScoreCandidate
- `internal/graph/` — topic DAG loader + scope resolution

### Phase 4 — Repository Layer
- `internal/repository/problem_repo.go` — CRUD + ANN search (`embedding <=>`)
- `internal/repository/user_repo.go`
- `internal/repository/session_repo.go` — append response tx
- `internal/repository/stats_repo.go`
- `internal/repository/problem_stats_repo.go`

### Phase 5 — Session Service
- `internal/services/practice_cache.go`
- `internal/services/practice_analytics.go` — mastery score, DNA EMA
- `internal/services/practice_session.go` — full state machine (Start/Submit/Skip/Close)
- `internal/services/graph_service.go` — topic-scope resolution

### Phase 6 — HTTP Layer
- `internal/handlers/practice_handler.go` — all practice endpoints
- `internal/handlers/health_handler.go`
- `internal/middleware/` — rate limiter + auth stub
- `cmd/server/main.go` — full wiring + graceful shutdown + stale session cleanup

### Phase 7 — Docker + Docs
- `Dockerfile` (multi-stage, ONNX Runtime)
- `docker-compose.yml`
- `.env.example`
- `README.md` — seed + run instructions

---

## 17. Key Design Decisions

### 17.1 100% Go — No Python
The Python embedding scripts (`generate_embeddings.py`, `seed.py`) are replaced by a Go pipeline using ONNX Runtime CGO bindings. The entire system — seeding, serving, scoring — ships as one binary.

### 17.2 Concept DNA via Embeddings
`embed_text` is constructed as:
```
"[PROBLEM] {name} [TOPIC] {topic} [SUBTOPIC] {subtopic} [TAGS] {tags} [SOURCE] {source}"
```
This means the 384-dim vector captures the full "conceptual DNA" — algorithmic mechanics, difficulty archetype, platform context. A LeetCode Two Sum variant and a Codeforces implementation problem cluster together if they share algorithmic core.

### 17.3 Glicko Scale Already Aligned with CF
Codeforces uses a ~800–3500 rating scale that is Glicko-derived. CF difficulty ratings can be used directly as initial `glicko_rating` values — no IRT calibration notebook needed. LeetCode and AtCoder are normalized into this range.

### 17.4 pgvector HNSW for Concept Similarity (Core Feature)
After a wrong answer, instead of loading all 18,000 problems into Go memory (O(N)):
```sql
SELECT * FROM problems 
WHERE topic = $1
ORDER BY embedding <=> $2   -- cosine ANN via HNSW index
LIMIT 30
```
This is O(log N) and returns the 30 most conceptually similar problems to the one the user just failed — the mathematical heart of the "Similar Concept Mapping" feature mentioned in the Notion workspace.

### 17.5 Future Extension Points
The binary is architected to add without refactoring:
- Auth service (`internal/handlers/auth_handler.go`)
- Roadmap service (`internal/services/roadmap_service.go`)  
- LLM integration (`internal/services/llm_service.go`)
- Compiler proxy (`internal/handlers/execute_handler.go`)
- Diagnostic quiz (`mode="DIAGNOSTIC"` in session start)

---

## 18. Judge0 Integration Design

The `is_correct` field in `POST /api/v1/practice/submit` is **never self-reported**. It is derived from a **Judge0 verdict**. The client submits code, gets a `judge0_token`, then calls the Transverse submit endpoint with that token.

### Flow
```
Client                     Backend                      Judge0
  │── POST /execute ────────►│── POST /submissions ──────►│
  │   {code, lang, prob_id}  │   {source_code, lang_id,   │
  │                          │    stdin (test cases)}      │
  │◄── {judge0_token} ───────│◄── {token} ────────────────│
  │                          │                            │
  │── POST /practice/submit ►│── GET /submissions/{token}►│
  │   {session_id,           │◄── {verdict, time, memory} │
  │    problem_id,           │                            │
  │    judge0_token,         │  status_id==3 → Accepted   │
  │    time_taken_ms}        │  else         → Wrong       │
  │◄── {is_correct,          │                            │
  │     verdict_detail,      │                            │
  │     next_problem, θ}     │                            │
```

### Submit DTO (updated for Judge0)
```go
// POST /api/v1/practice/submit
type SubmitRequest struct {
    SessionID   string `json:"session_id"`
    ProblemID   string `json:"problem_id"`
    Judge0Token string `json:"judge0_token"`  // token from Judge0 submission
    TimeTakenMs int64  `json:"time_taken_ms"` // wall-clock time user spent thinking
}

type SubmitResponse struct {
    IsCorrect     bool            `json:"is_correct"`
    VerdictDetail VerdictDetail   `json:"verdict_detail"`
    ThetaBefore   float64         `json:"theta_before"`
    ThetaAfter    float64         `json:"theta_after"`
    NextProblem   *ProblemPayload `json:"next_problem,omitempty"`
}

type VerdictDetail struct {
    StatusID   int    `json:"status_id"`           // Judge0 status code
    StatusDesc string `json:"status_desc"`          // "Accepted", "Wrong Answer", etc.
    TimeMs     int    `json:"time_ms"`              // actual execution time
    MemoryKB   int    `json:"memory_kb"`
    Stderr     string `json:"stderr,omitempty"`
    CompileOut string `json:"compile_out,omitempty"`
}
```

### Judge0 Status → Engine Behavior
```
status_id == 3 (Accepted)            → is_correct=true, full theta update, afterCorrect picker
status_id == 4 (Wrong Answer)        → is_correct=false, full theta update, afterWrong picker
status_id == 5 (Time Limit Exceeded) → is_correct=false, full theta update, afterWrong picker
                                        (TLE = right concept, wrong complexity → similar problem)
status_id == 6 (Runtime Error)       → is_correct=false, full theta update, afterWrong picker
status_id == 7 (Compilation Error)   → NO theta update, NO attempt recorded, re-serve same problem
status_id == 8 (Memory Limit)        → is_correct=false, full theta update, afterWrong picker
```

### Compile Error Special Case
CE is a **syntax error, not a knowledge gap**:
- Do NOT update theta
- Do NOT record attempt in `user_problem_stats`
- Re-serve the same problem as `next_problem`
- Return `is_correct: false` + `verdict_detail.status_desc = "Compilation Error"`

### Additional Execution Endpoints
```
POST /api/v1/execute
     Body: {problem_id, language_id, source_code, custom_stdin?}
     → Submits to Judge0, returns {judge0_token}

GET  /api/v1/execute/{judge0_token}
     → Polls Judge0, returns {status_id, status_desc, stdout, stderr, time_ms, memory_kb}
```

### Supported Languages (Judge0 IDs)
```go
var SupportedLanguages = map[string]int{
    "c":    50,   // C (GCC 9.2.0)
    "cpp":  54,   // C++ (GCC 9.2.0)
    "java": 62,   // Java (OpenJDK 13)
    "py":   71,   // Python 3
    "js":   63,   // Node.js 12
    "go":   60,   // Go 1.13
    "rust": 73,   // Rust 1.40
}
```

### Config Additions
```
JUDGE0_BASE_URL   = "https://judge0-ce.p.rapidapi.com"  # or self-hosted
JUDGE0_API_KEY    = ""
JUDGE0_TIMEOUT_MS = 5000
```

---

## 19. Out of Scope (Future Work)

- Auth / OAuth2 / JWT (handler stub only for now)
- LLM roadmap generation
- Compiler integration
- Frontend (Next.js)
- Redis pub/sub
- DKVMN deep knowledge tracing (future ML upgrade, see `reference_docs/dkvmn.md`)
- Leaderboard / contest modes

---

## 19. Success Criteria

1. `go run ./pipeline/seed` seeds all 18,000+ problems with embeddings in < 30 minutes
2. `POST /api/v1/practice/start` returns first problem in < 50ms
3. `POST /api/v1/practice/submit` computes next problem (incl. pgvector ANN) in < 100ms
4. Concept similarity works: "Two Sum" → surfaces "3Sum", "Container With Most Water", "Pair Sum" from multiple platforms
5. After 10 correct answers in Binary Search, user theta increases by 200–300 Glicko points
6. After 3 wrong DP answers, next problems are DP at a lower Glicko rating
7. `docker-compose up` brings up the full stack with zero manual setup

---

*Plan v1.0 — Awaiting approval before implementation begins.*
