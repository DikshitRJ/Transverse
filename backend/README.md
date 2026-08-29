# Transverse — DSA/CP Adaptive Heuristic Engine Backend

Transverse is a high-performance, deterministic adaptive learning engine for Competitive Programming (CP) and Data Structures & Algorithms (DSA). Designed as a "Duolingo for DSA/CP", Transverse combines **Item Response Theory (IRT 1PL Rasch)**, a **7-factor deterministic heuristic scoring model**, **Glicko-2 psychometric ratings**, and **384-dimensional vector embeddings** to deliver personalized problem recommendations in real time.

---

## 🏛️ Architecture Overview

```
                          ┌───────────────────────────┐
                          │   HTTP Client / Frontend  │
                          └─────────────┬─────────────┘
                                        │ (JWT / REST API)
                                        ▼
                          ┌───────────────────────────┐
                          │  Chi v5 HTTP Router       │
                          │  - Auth & Rate Limiting   │
                          └─────────────┬─────────────┘
                                        │
                 ┌──────────────────────┼──────────────────────┐
                 ▼                      ▼                      ▼
        ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐
        │  PracticeHandler │  │   UserHandler    │  │  HealthHandler   │
        └────────┬─────────┘  └────────┬─────────┘  └────────┬─────────┘
                 │                     │                     │
                 ▼                     │                     │
        ┌──────────────────┐           │                     │
        │  PracticeService │           │                     │
        └────────┬─────────┘           │                     │
                 │                     │                     │
      ┌──────────┴───────────────┬─────┴──────────────┐      │
      ▼                          ▼                    ▼      ▼
┌──────────────┐         ┌───────────────┐     ┌───────────────────┐
│  7-Factor    │         │  IRT Theta &  │     │   Repository &    │
│  Scorer      │         │  Glicko-2     │     │   Cache Layer     │
└──────────────┘         └───────────────┘     └─────────┬─────────┘
                                                         │
                                    ┌────────────────────┴────────────────────┐
                                    ▼                                         ▼
                         ┌──────────────────────┐                  ┌──────────────────────┐
                         │ PostgreSQL 16 +      │                  │ In-Memory LRU Cache  │
                         │ pgvector (HNSW Index)│                  │ (TTL-backed)         │
                         └──────────────────────┘                  └──────────────────────┘
```

### The 7-Factor Heuristic Scorer

Candidate problems are scored and ranked deterministically using a context-aware 7-factor model:

| Factor | Description | Weight (After Correct) | Weight (After Wrong) |
|---|---|---|---|
| **DifficultyFit** | Distance between effective user ability $\theta_{\text{eff}}$ and problem difficulty $R_p$: $1 - \frac{\|\theta_{\text{eff}} - R_p\|}{1500}$ | 50% | 15% |
| **ConceptSimilarity** | Vector cosine similarity between problem embeddings ($384$-dim BAAI/bge-small-en-v1.5) | 15% | 25% |
| **TopicProgression** | Relevance bonus for active roadmap topics & prerequisite dependencies | 10% | 5% |
| **NoveltyFactor** | Repetition penalty decaying with previous attempts: $1 - \min(\frac{\text{attempts}}{5}, 0.5)$ | 10% | 5% |
| **ImmediateReinforce** | Reinforcement weight targeting concept-similar problems at calibrated difficulty | 5% | 35% |
| **PlatformDiversity** | Entropy bonus balancing sources (LeetCode, Codeforces, AtCoder, CSES) | 5% | 5% |
| **CarelessnessPenalty** | Subtracted penalty based on error rate on easier problems | 5% | 10% |

### Effective Theta Calculation
$$\theta_{\text{eff}} = \theta_{\text{current}} + (\text{TopicBias} \times 200) + \text{Momentum} + \text{PhaseAdjustment}$$

---

## 🚀 Prerequisites

- **Docker** & **Docker Compose** (recommended for production and local development)
- **Go 1.23+** (if building from source)
- **PostgreSQL 16** with `pgvector` extension
- **ONNX Runtime shared libraries** (`libonnxruntime.so.1.18.1`) for Go CGO embeddings

---

## ⚡ Quick Start with Docker Compose

1. Clone the repository and navigate to the backend directory:
   ```bash
   cd backend
   ```

2. Start the PostgreSQL + pgvector database and backend server:
   ```bash
   docker-compose up --build -d
   ```

3. Verify service health:
   ```bash
   curl http://localhost:8080/health
   ```
   **Response:**
   ```json
   {
     "status": "ok",
     "pool_total": 4,
     "pool_idle": 4,
     "pool_acquired": 0
   }
   ```

---

## ⚙️ Configuration & Environment Variables

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | HTTP server listening port |
| `DATABASE_URL` | `postgres://transverse:transverse@localhost:5432/transverse` | PostgreSQL connection URL |
| `DB_POOL_MIN_CONNS` | `4` | Minimum database connections in pool |
| `DB_POOL_MAX_CONNS` | `20` | Maximum database connections in pool |
| `ONNX_MODEL_PATH` | `./models/bge-small-en-v1.5.onnx` | Path to the ONNX embedding model |
| `TOPICS_GRAPH_PATH` | `../data/topics.json` | Path to the topic knowledge graph DAG |
| `JUDGE0_BASE_URL` | `https://judge0-ce.p.rapidapi.com` | Judge0 code execution API base URL |
| `JUDGE0_API_KEY` | `""` | Judge0 / RapidAPI authentication key |
| `JUDGE0_TIMEOUT_MS` | `5000` | Judge0 HTTP timeout in milliseconds |
| `CACHE_ENABLED` | `true` | Toggles in-memory caching |
| `JWT_SECRET` | `change-me-in-production` | Secret key for JWT verification |
| `BYPASS_AUTH` | `false` | When `true`, automatically authenticates with `dev-user-001` |

---

## 📚 API Reference

All `/api/v1/*` endpoints accept an optional `Authorization: Bearer <token>` header. When `BYPASS_AUTH=true`, requests without tokens default to user `dev-user-001`.

### 1. Health Check
`GET /health`
```json
{
  "status": "ok",
  "pool_total": 4,
  "pool_idle": 4,
  "pool_acquired": 0
}
```

---

### 2. Practice Engine Endpoints

#### Start Practice Session
`POST /api/v1/practice/start`

**Request:**
```json
{
  "mode": "ADAPTIVE",
  "scope": {
    "topics": ["arrays_and_hashing", "two_pointers"],
    "sources": ["leetcode", "codeforces"],
    "difficulty_range": [1000, 1600]
  }
}
```

**Response:**
```json
{
  "session_id": "sess_18e2a47f9c2d",
  "mode": "ADAPTIVE",
  "theta": 1500.0,
  "current_problem": {
    "id": "leetcode-1",
    "source": "leetcode",
    "name": "Two Sum",
    "url": "https://leetcode.com/problems/two-sum",
    "slug": "two-sum",
    "tags": ["Array", "Hash Table"],
    "topic": "arrays_and_hashing",
    "subtopic": "hash_map",
    "difficulty_label": "easy",
    "solve_rate": 0.49,
    "avg_time_ms": 90000
  },
  "status": "ACTIVE",
  "created_at": "2026-08-30T01:50:00Z"
}
```

---

#### Submit Problem Answer
`POST /api/v1/practice/submit`

**Request:**
```json
{
  "session_id": "sess_18e2a47f9c2d",
  "judge0_token": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "time_taken_ms": 75000
}
```

**Response:**
```json
{
  "is_correct": true,
  "verdict": {
    "status_id": 3,
    "status_desc": "Accepted",
    "time_ms": 42,
    "memory_kb": 3480
  },
  "theta_before": 1500.0,
  "theta_after": 1528.4,
  "next_problem": {
    "id": "cf-4-A",
    "source": "codeforces",
    "name": "Watermelon",
    "url": "https://codeforces.com/problemset/problem/4/A",
    "slug": "watermelon",
    "tags": ["brute force", "math"],
    "topic": "math_and_number_theory",
    "subtopic": "basic_math",
    "difficulty_label": "easy",
    "solve_rate": 0.82,
    "avg_time_ms": 60000
  },
  "session_status": "ACTIVE",
  "question_count": 1,
  "carelessness_penalty": 0.0
}
```

---

#### Skip Problem
`POST /api/v1/practice/skip`

**Request:**
```json
{
  "session_id": "sess_18e2a47f9c2d",
  "time_taken_ms": 30000
}
```

**Response:**
```json
{
  "skipped": true,
  "theta_before": 1528.4,
  "theta_after": 1505.1,
  "next_problem": {
    "id": "leetcode-167",
    "source": "leetcode",
    "name": "Two Sum II - Input Array Is Sorted",
    "url": "https://leetcode.com/problems/two-sum-ii-input-array-is-sorted",
    "slug": "two-sum-ii-input-array-is-sorted",
    "tags": ["Array", "Two Pointers", "Binary Search"],
    "topic": "two_pointers",
    "subtopic": "sorted_two_sum",
    "difficulty_label": "medium",
    "solve_rate": 0.61,
    "avg_time_ms": 110000
  },
  "question_count": 2
}
```

---

#### Close Session
`POST /api/v1/practice/close`

**Request:**
```json
{
  "session_id": "sess_18e2a47f9c2d"
}
```

**Response:**
```json
{
  "session_id": "sess_18e2a47f9c2d",
  "status": "COMPLETED",
  "theta_start": 1500.0,
  "theta_final": 1550.2,
  "mastery_score": 16.7,
  "accuracy": 0.8,
  "total_questions": 5,
  "total_solved": 4,
  "per_topic_breakdown": {
    "arrays_and_hashing": {
      "topic": "arrays_and_hashing",
      "mastery_score": 16.7,
      "theta": 1550.2,
      "glicko_rating": 1542.8,
      "attempt_count": 3,
      "correct_count": 3
    }
  }
}
```

---

#### Get Session Status
`GET /api/v1/practice/session/{id}`

---

#### Find Semantically Similar Problems (pgvector ANN)
`GET /api/v1/practice/similar?problem_id=leetcode-1&limit=5`

**Response:**
```json
{
  "problem_id": "leetcode-1",
  "similar_problems": [
    {
      "id": "leetcode-167",
      "source": "leetcode",
      "name": "Two Sum II - Input Array Is Sorted",
      "url": "https://leetcode.com/problems/two-sum-ii-input-array-is-sorted",
      "slug": "two-sum-ii-input-array-is-sorted",
      "tags": ["Array", "Two Pointers"],
      "topic": "two_pointers",
      "subtopic": "sorted_two_sum",
      "difficulty_label": "medium",
      "solve_rate": 0.61,
      "avg_time_ms": 110000
    }
  ]
}
```

---

#### List Curriculum Topics & Mastery
`GET /api/v1/practice/topics`

---

### 3. Code Execution Endpoints (Judge0 Proxy)

#### Execute Code
`POST /api/v1/execute`

**Request:**
```json
{
  "problem_id": "leetcode-1",
  "language_id": 71,
  "source_code": "def twoSum(nums, target):\n    seen = {}\n    for i, n in enumerate(nums):\n        if target - n in seen:\n            return [seen[target - n], i]\n        seen[n] = i\n",
  "custom_stdin": "[2,7,11,15]\n9"
}
```

**Response:**
```json
{
  "judge0_token": "e30e6618-fa0f-4886-9a57-fb92789182aa"
}
```

#### Poll Execution Verdict
`GET /api/v1/execute/{token}`

**Response:**
```json
{
  "token": "e30e6618-fa0f-4886-9a57-fb92789182aa",
  "status_id": 3,
  "status_desc": "Accepted",
  "time_ms": 38,
  "memory_kb": 4120,
  "is_done": true
}
```

---

### 4. Problem Search
`GET /api/v1/problems/search?q=two+sum&topic=arrays_and_hashing&source=leetcode&limit=10`

---

### 5. User Profile & History

#### Get User Profile & LearningDNA
`GET /api/v1/user/profile`

**Response:**
```json
{
  "id": "dev-user-001",
  "username": "dev-user",
  "email": "dev-user@transverse.local",
  "theta": 1550.2,
  "glicko_rating": 1542.8,
  "glicko_rd": 312.4,
  "dna": {
    "avg_accuracy": 0.8,
    "avg_time_taken_ms": 78000,
    "avg_solve_velocity": 4.5,
    "carelessness_index": 0.05,
    "peak_performance_hour": 18,
    "avg_session_length": 5.0,
    "total_sessions": 3,
    "total_problems_solved": 12,
    "topic_bias": {
      "arrays_and_hashing": 0.12
    },
    "preferred_platform": "leetcode",
    "streak_record": 7
  },
  "created_at": "2026-08-30T01:40:00Z"
}
```

#### Get User Practice History
`GET /api/v1/user/history?limit=10&offset=0`

---

## 🛠️ Data Seeding Pipeline

To embed and seed scraped competitive programming datasets into PostgreSQL with vector embeddings:

```bash
# Run the Go native ONNX seeder
go run ./pipeline/seed \
  --data-dir ./data/generated \
  --model ./models/bge-small-en-v1.5.onnx \
  --db "postgres://transverse:transverse@localhost:5432/transverse" \
  --batch-size 32 \
  --migrate
```

---

## 🧪 Testing

Run all unit and integration tests across scoring, IRT, Glicko, and services:

```bash
go test -v -race ./...
```
