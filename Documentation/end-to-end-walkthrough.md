# End-to-End Walkthrough

Follow this sequence to simulate the complete user journey via HTTP calls. Replace `${TOKEN}` with your generated access token.

---

### 1. Login & Identity
Authenticate via OAuth (`/api/v1/auth/oauth/github/redirect` or test tokens) and export the bearer token:

```bash
export TOKEN="your_jwt_here"
```

---

### 2. Fetch Dynamic Progressive Roadmap (On Frontend Load)
Whenever the frontend loads, it calls `GET /api/v1/roadmap` (or `GET /api/v1/roadmap/me`).

```bash
curl -X GET http://localhost:8080/api/v1/roadmap \
  -H "Authorization: Bearer $TOKEN"
```

**Key Behavior**:
- Returns **only the single current active section** with all its subsections (nodes), including:
  - Topic ability rating and normalized 0-100% mastery score.
  - Curated tutorials with reading times, difficulty, source links, and summaries.
  - Practice questions with statements, sample test cases, difficulty ratings, and starter code templates for 8 languages.
- Upcoming sections are locked previews (`status: "LOCKED"`).

---

### 3. Complete a Roadmap Node or Bypass via Test-Out

```bash
# Mark a subsection/node complete after reading tutorial / solving problems:
curl -X POST http://localhost:8080/api/v1/roadmap/nodes/<node_id>/complete \
  -H "Authorization: Bearer $TOKEN"

# Or test out of a node directly:
curl -X POST http://localhost:8080/api/v1/roadmap/nodes/<node_id>/test-out \
  -H "Authorization: Bearer $TOKEN"
```
*When all nodes in Section 1 are mastered/tested-out, Section 2 is automatically unlocked and generated.*

---

### 4. Scrape Problem & Generate Templates on Demand
If a problem URL from LeetCode or Codeforces is provided:

```bash
curl -X POST http://localhost:8080/api/v1/problems/scrape \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"url": "https://leetcode.com/problems/two-sum/"}'
```
*Expected: Returns problem statement HTML, sample input/output test cases, time/memory limits, and starter templates for Python, C++, Java, JS, Go, Rust, C, Kotlin.*

---

### 5. Multi-Test-Case Batch Execution via Local Judge0
Execute user source code against multiple test cases in a single call:

```bash
curl -X POST http://localhost:8080/api/v1/execute/batch \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "language_id": 71,
    "source_code": "import sys\nlines = sys.stdin.read().split()\nprint(lines[0])\n",
    "test_cases": [
      {"input": "hello", "output": "hello"},
      {"input": "world", "output": "world"}
    ]
  }'
```
*Expected: 200 OK with `all_passed: true`, `passed_count: 2`, `total_count: 2`, `overall_status: "Accepted"`, max time/memory, and per-test-case stdout/stderr.*

---

### 6. Adaptive Practice Loop & AI Hints

```bash
# 1. Start adaptive practice session
curl -X POST http://localhost:8080/api/v1/practice/start \
  -H "Authorization: Bearer $TOKEN"

# 2. Submit solution
curl -X POST http://localhost:8080/api/v1/practice/submit \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "session_id": "<session_id>",
    "problem_id": "<problem_id>",
    "source_code": "...",
    "language_id": 71,
    "time_taken_ms": 45000
  }'

# 3. Request LLM hint on struggle
curl -X POST http://localhost:8080/api/v1/practice/<session_id>/hint \
  -H "Authorization: Bearer $TOKEN"
```
