# End-to-End Walkthrough

Follow this sequence to simulate the complete user journey via HTTP calls. Replace `${TOKEN}` with your generated access token.

### 1. Login
To trigger OAuth logic, direct the user to `/api/v1/auth/oauth/github/redirect`. Upon return, you can use the token.
*(For local testing without a frontend, you can bypass if a test endpoint generates tokens, but otherwise follow the redirect to get your Bearer token).*

```bash
export TOKEN="your_jwt_here"
```

### 2. Submit Evidence
Upload external references (e.g. GitHub) to generate a skill baseline.

```bash
curl -X POST http://localhost:8080/api/v1/evidence/github \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"username": "octocat"}'
```
*Expected: 202 Accepted, returns `job_id`.*

Wait for the background job to finish. You can listen via SSE or poll:
```bash
curl -X GET http://localhost:8080/api/v1/jobs/<job_id> \
  -H "Authorization: Bearer $TOKEN"
```

### 3. Generate Hypotheses
Use the gathered evidence to trigger LLM analysis for skill hypothesis generation.

```bash
curl -X POST http://localhost:8080/api/v1/hypotheses/generate \
  -H "Authorization: Bearer $TOKEN"
```
*Expected: 202 Accepted. Poll job until completion.*

### 4. Run Verification Quiz
Start a quiz session to evaluate the generated hypotheses deterministically.

```bash
curl -X POST http://localhost:8080/api/v1/quiz/verification/start \
  -H "Authorization: Bearer $TOKEN"
```
*(Returns quiz session details. Submit answers via `/api/v1/quiz/{sessionId}/answer`, then complete the session).*

### 5. Generate Roadmap
Once verification resolves your skill baseline, generate a roadmap targeting a specific role.

```bash
curl -X POST http://localhost:8080/api/v1/roadmap/generate \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"target_role": "SDE Interview Prep"}'
```

### 6. Practice & Hint Request
Start practicing problems on nodes you've unlocked in your roadmap. If you get stuck, request a hint.

```bash
# Start practice session
curl -X POST http://localhost:8080/api/v1/practice/start \
  -H "Authorization: Bearer $TOKEN"

# If a submission fails and you need a hint:
curl -X POST http://localhost:8080/api/v1/practice/<session_id>/hint \
  -H "Authorization: Bearer $TOKEN"
```
*Expected: 202 Accepted. The LLM hint job runs async and will be pushed via SSE once ready.*
