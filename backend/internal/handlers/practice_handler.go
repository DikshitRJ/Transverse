package handlers

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"transverse/internal/middleware"
	"transverse/internal/models"
	"transverse/internal/repository"
	"transverse/internal/scraper"
	"transverse/internal/services"
)

// PracticeHandler exposes HTTP endpoints for adaptive sessions, submissions, code execution, and search.
type PracticeHandler struct {
	practice    *services.PracticeService
	judge0      *services.Judge0Service
	problemRepo *repository.ProblemRepo
	scraper     *scraper.UnifiedScraper
}

// NewPracticeHandler constructs a new PracticeHandler instance.
func NewPracticeHandler(
	practice *services.PracticeService,
	judge0 *services.Judge0Service,
	problemRepo *repository.ProblemRepo,
	scraper *scraper.UnifiedScraper,
) *PracticeHandler {
	return &PracticeHandler{
		practice:    practice,
		judge0:      judge0,
		problemRepo: problemRepo,
		scraper:     scraper,
	}
}

// StartSession initiates or resumes an adaptive practice session.
// POST /api/v1/practice/start
func (h *PracticeHandler) StartSession(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok || userID == "" {
		userID = "dev-user-001"
	}

	var req models.StartSessionRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	startReq := services.StartRequest{
		UserID:          userID,
		Mode:            req.Mode,
		Topics:          req.Scope.Topics,
		Subtopics:       req.Scope.Subtopics,
		Sources:         req.Scope.Sources,
		DifficultyRange: req.Scope.DifficultyRange,
	}

	resp, err := h.practice.Start(r.Context(), startReq)
	if err != nil {
		slog.Error("failed to start practice session", "user_id", userID, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to start session: "+err.Error())
		return
	}

	var currentProblem *models.ProblemPayload
	if resp.FirstProblem != nil {
		cp := models.ToProblemPayload(resp.FirstProblem)
		currentProblem = &cp
	}

	writeJSON(w, http.StatusOK, models.StartSessionResponse{
		SessionID:      resp.Session.ID,
		Mode:           resp.Session.Mode,
		Theta:          resp.ThetaStart,
		CurrentProblem: currentProblem,
		Status:         resp.Session.Status,
		CreatedAt:      resp.Session.CreatedAt,
	})
}

type submitPayload struct {
	SessionID   string `json:"session_id"`
	ProblemID   string `json:"problem_id,omitempty"`
	Judge0Token string `json:"judge0_token"`
	TimeTakenMs int64  `json:"time_taken_ms"`
}

// SubmitAnswer evaluates a submission against Judge0, updates ability, and returns the next problem.
// POST /api/v1/practice/submit
func (h *PracticeHandler) SubmitAnswer(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok || userID == "" {
		userID = "dev-user-001"
	}

	var payload submitPayload
	if !decodeJSON(w, r, &payload) {
		return
	}

	if payload.SessionID == "" {
		writeError(w, http.StatusBadRequest, "session_id is required")
		return
	}

	if payload.Judge0Token == "" {
		writeError(w, http.StatusBadRequest, "judge0_token is required")
		return
	}

	submitReq := services.SubmitRequest{
		UserID:      userID,
		SessionID:   payload.SessionID,
		ProblemID:   payload.ProblemID,
		Judge0Token: payload.Judge0Token,
		TimeTakenMs: payload.TimeTakenMs,
	}
	resp, err := h.practice.Submit(r.Context(), submitReq)
	if err != nil {
		slog.Error("failed to submit answer", "user_id", userID, "session_id", payload.SessionID, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to submit answer: "+err.Error())
		return
	}

	var np *models.ProblemPayload
	if resp.NextProblem != nil {
		p := models.ToProblemPayload(resp.NextProblem)
		np = &p
	}

	writeJSON(w, http.StatusOK, models.SubmitResponse{
		IsCorrect: resp.IsCorrect,
		Verdict: models.VerdictDetail{
			StatusID:      resp.Verdict.StatusID,
			StatusDesc:    resp.Verdict.StatusDesc,
			TimeMs:        resp.Verdict.TimeMs,
			MemoryKB:      resp.Verdict.MemoryKB,
			Stderr:        resp.Verdict.Stderr,
			CompileOutput: resp.Verdict.CompileOut,
		},
		ThetaBefore:   resp.ThetaBefore,
		ThetaAfter:    resp.ThetaAfter,
		NextProblem:   np,
		QuestionCount: resp.QuestionCount,
		SessionStatus: resp.SessionStatus,
	})
}

type skipPayload struct {
	SessionID   string `json:"session_id"`
	ProblemID   string `json:"problem_id,omitempty"`
	TimeTakenMs int64  `json:"time_taken_ms"`
}

// SkipProblem skips the current problem in an active session and fetches the next candidate.
// POST /api/v1/practice/skip
func (h *PracticeHandler) SkipProblem(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok || userID == "" {
		userID = "dev-user-001"
	}

	var payload skipPayload
	if !decodeJSON(w, r, &payload) {
		return
	}

	if payload.SessionID == "" {
		writeError(w, http.StatusBadRequest, "session_id is required")
		return
	}

	resp, err := h.practice.Skip(r.Context(), userID, payload.SessionID, payload.ProblemID, payload.TimeTakenMs)
	if err != nil {
		slog.Error("failed to skip problem", "user_id", userID, "session_id", payload.SessionID, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to skip problem: "+err.Error())
		return
	}

	var np *models.ProblemPayload
	if resp.NextProblem != nil {
		p := models.ToProblemPayload(resp.NextProblem)
		np = &p
	}

	writeJSON(w, http.StatusOK, models.SkipResponse{
		Skipped:       true,
		ThetaBefore:   resp.ThetaBefore,
		ThetaAfter:    resp.ThetaAfter,
		NextProblem:   np,
		QuestionCount: resp.QuestionCount,
	})
}

type closePayload struct {
	SessionID string `json:"session_id"`
}

// CloseSession ends an active session and calculates final Glicko-2 ratings and LearningDNA.
// POST /api/v1/practice/close
func (h *PracticeHandler) CloseSession(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok || userID == "" {
		userID = "dev-user-001"
	}

	var payload closePayload
	if !decodeJSON(w, r, &payload) {
		return
	}

	if payload.SessionID == "" {
		writeError(w, http.StatusBadRequest, "session_id is required")
		return
	}

	resp, err := h.practice.Close(r.Context(), userID, payload.SessionID)
	if err != nil {
		slog.Error("failed to close session", "user_id", userID, "session_id", payload.SessionID, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to close session: "+err.Error())
		return
	}

	topicBreakdown := make(map[string]models.TopicProgress)
	for k, v := range resp.TopicBreakdown {
		topicBreakdown[k] = models.TopicProgress{
			Topic:        v.Topic,
			MasteryScore: v.MasteryScore,
		}
	}

	writeJSON(w, http.StatusOK, models.CloseSessionResponse{
		SessionID:         resp.SessionID,
		Status:            "CLOSED",
		ThetaStart:        resp.ThetaStart,
		ThetaFinal:        resp.ThetaFinal,
		MasteryScore:      resp.MasteryScore,
		Accuracy:          resp.Accuracy,
		TotalQuestions:    resp.TotalProblems,
		TotalSolved:       resp.CorrectCount,
		PerTopicBreakdown: topicBreakdown,
	})
}

// GetSession retrieves the complete details of a specific practice session.
// GET /api/v1/practice/session/{id}
func (h *PracticeHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok || userID == "" {
		userID = "dev-user-001"
	}

	sessionID := chi.URLParam(r, "id")
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "session id parameter is required")
		return
	}

	session, currentProblem, err := h.practice.GetSession(r.Context(), userID, sessionID)
	if err != nil {
		slog.Error("failed to get session", "user_id", userID, "session_id", sessionID, "error", err)
		writeError(w, http.StatusNotFound, "session not found: "+err.Error())
		return
	}

	var cp *models.ProblemPayload
	if currentProblem != nil {
		p := models.ToProblemPayload(currentProblem)
		cp = &p
	}
	responses, _ := session.Responses()
	scope, _ := session.Scope()

	writeJSON(w, http.StatusOK, models.GetSessionResponse{
		SessionID:      session.ID,
		UserID:         session.UserID,
		Mode:           session.Mode,
		Status:         session.Status,
		Scope:          scope,
		ThetaStart:     session.ThetaStart,
		ThetaCurrent:   session.ThetaCurrent,
		QuestionCount:  session.QuestionCount,
		CurrentProblem: cp,
		Responses:      responses,
		CreatedAt:      session.CreatedAt,
		UpdatedAt:      session.UpdatedAt,
	})
}

type hintPayload struct {
	HintLevel int `json:"hint_level"`
}

// RequestHint requests the next-level LLM hint for a practice session.
// POST /api/v1/practice/{id}/hint
func (h *PracticeHandler) RequestHint(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok || userID == "" {
		userID = "dev-user-001"
	}

	sessionID := chi.URLParam(r, "id")
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "session id parameter is required")
		return
	}

	var payload hintPayload
	if r.ContentLength > 0 {
		_ = decodeJSON(w, r, &payload)
	}
	if payload.HintLevel <= 0 {
		payload.HintLevel = 1
	}

	jobID, err := h.practice.RequestHint(r.Context(), userID, sessionID, payload.HintLevel)
	if err != nil {
		slog.Error("failed to request hint", "user_id", userID, "session_id", sessionID, "error", err)
		if strings.Contains(err.Error(), "rate limit") {
			writeError(w, http.StatusTooManyRequests, err.Error())
		} else {
			writeError(w, http.StatusInternalServerError, "failed to request hint: "+err.Error())
		}
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{
		"job_id": jobID,
	})
}

// GetErrorAnalysis retrieves the latest LLM error analysis for the session.
// GET /api/v1/practice/{id}/error-analysis
func (h *PracticeHandler) GetErrorAnalysis(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok || userID == "" {
		userID = "dev-user-001"
	}

	sessionID := chi.URLParam(r, "id")
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "session id parameter is required")
		return
	}

	analysis, err := h.practice.GetErrorAnalysis(r.Context(), userID, sessionID)
	if err != nil {
		slog.Error("failed to get error analysis", "user_id", userID, "session_id", sessionID, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get error analysis: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, analysis)
}

// GetSimilar returns nearest-neighbor problems using cosine embedding similarity.
// GET /api/v1/practice/similar?problem_id=X&limit=5
func (h *PracticeHandler) GetSimilar(w http.ResponseWriter, r *http.Request) {
	problemID := r.URL.Query().Get("problem_id")
	if problemID == "" {
		writeError(w, http.StatusBadRequest, "problem_id query parameter is required")
		return
	}

	limit := 5
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}

	topic := r.URL.Query().Get("topic")
	similar, err := h.practice.GetSimilar(r.Context(), problemID, topic, limit)
	if err != nil {
		slog.Error("failed to find similar problems", "problem_id", problemID, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to find similar problems: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, models.SimilarProblemsResponse{
		ProblemID:       problemID,
		SimilarProblems: models.ToProblemPayloads(similar),
	})
}

// GetTopics returns all curriculum topics along with the authenticated user's mastery scores.
// GET /api/v1/practice/topics
func (h *PracticeHandler) GetTopics(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok || userID == "" {
		userID = "dev-user-001"
	}

	topics, err := h.practice.GetTopics(r.Context(), userID)
	if err != nil {
		slog.Error("failed to get topics", "user_id", userID, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get topics: "+err.Error())
		return
	}

	progress := make([]models.TopicProgress, len(topics))
	for i, t := range topics {
		progress[i] = models.TopicProgress{
			Topic:        t.Topic,
			MasteryScore: t.MasteryScore,
			Theta:        t.Theta,
			GlickoRating: t.GlickoRating,
			AttemptCount: t.AttemptCount,
			CorrectCount: t.CorrectCount,
		}
	}
	writeJSON(w, http.StatusOK, models.TopicsResponse{
		Topics: progress,
	})
}

// Execute submits code to Judge0 asynchronously and immediately returns the tracking token.
// POST /api/v1/execute
func (h *PracticeHandler) Execute(w http.ResponseWriter, r *http.Request) {
	var req models.ExecuteRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	stdin := req.CustomStdin
	expectedOutput := ""

	if req.ProblemID != "" && stdin == "" {
		if problem, err := h.problemRepo.GetByID(r.Context(), req.ProblemID); err == nil && problem != nil {
			if len(problem.TestCases) > 0 {
				stdin = problem.TestCases[0].Input
				expectedOutput = problem.TestCases[0].Output
			}
		}
	}

	token, err := h.judge0.SubmitCode(r.Context(), services.SubmitCodeRequest{
		SourceCode:     req.SourceCode,
		LanguageID:     req.LanguageID,
		Stdin:          stdin,
		ExpectedOutput: expectedOutput,
	})
	if err != nil {
		slog.Error("judge0 code submission failed", "error", err)
		writeError(w, http.StatusBadGateway, "execution service submission failed: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, models.ExecuteResponse{
		Judge0Token: token,
	})
}

// GetVerdict retrieves the current verdict and execution output for a submission token.
// GET /api/v1/execute/{token}
func (h *PracticeHandler) GetVerdict(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if token == "" {
		writeError(w, http.StatusBadRequest, "token path parameter is required")
		return
	}

	verdict, err := h.judge0.GetVerdict(r.Context(), token)
	if err != nil {
		slog.Error("judge0 get verdict failed", "token", token, "error", err)
		writeError(w, http.StatusBadGateway, "failed to retrieve execution verdict: "+err.Error())
		return
	}

	isDone := verdict.StatusID > 2 // 1: In Queue, 2: Processing, >2: Finished

	writeJSON(w, http.StatusOK, models.VerdictPollResponse{
		Token:         token,
		StatusID:      verdict.StatusID,
		StatusDesc:    verdict.StatusDesc,
		TimeMs:        verdict.TimeMs,
		MemoryKB:      verdict.MemoryKB,
		Stderr:        verdict.Stderr,
		CompileOutput: verdict.CompileOutput,
		Message:       verdict.Message,
		IsDone:        isDone,
	})
}

// SearchProblems filters problems across tags, platforms, ratings, and keywords.
// GET /api/v1/problems/search
func (h *PracticeHandler) SearchProblems(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	limit := 20
	offset := 0
	if lStr := q.Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}
	if limit > 100 {
		limit = 100
	}
	if oStr := q.Get("offset"); oStr != "" {
		if o, err := strconv.Atoi(oStr); err == nil && o >= 0 {
			offset = o
		}
	}

	req := models.ProblemSearchRequest{
		Query:           strings.TrimSpace(q.Get("q")),
		Topic:           strings.TrimSpace(q.Get("topic")),
		Source:          strings.TrimSpace(q.Get("source")),
		DifficultyLabel: strings.TrimSpace(q.Get("difficulty_label")),
		Limit:           limit,
		Offset:          offset,
	}

	resp, err := h.practice.SearchProblems(r.Context(), req)
	if err != nil {
		slog.Error("problem search failed", "error", err)
		writeError(w, http.StatusInternalServerError, "search failed: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// ExecuteBatch runs the given source code against multiple test cases and returns comprehensive verdicts.
// POST /api/v1/execute/batch
func (h *PracticeHandler) ExecuteBatch(w http.ResponseWriter, r *http.Request) {
	var req models.BatchExecuteRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	if strings.TrimSpace(req.SourceCode) == "" {
		writeError(w, http.StatusBadRequest, "source_code cannot be empty")
		return
	}

	if req.LanguageID <= 0 {
		writeError(w, http.StatusBadRequest, "valid language_id is required")
		return
	}

	// If no custom test cases were provided in request, load from problem entity
	if len(req.TestCases) == 0 && req.ProblemID != "" && h.problemRepo != nil {
		prob, err := h.problemRepo.GetByID(r.Context(), req.ProblemID)
		if err == nil && prob != nil {
			if len(prob.TestCases) > 0 {
				req.TestCases = prob.TestCases
			} else if prob.URL != "" && h.scraper != nil {
				// Scrape test cases on demand if none stored yet
				if scraped, sErr := h.scraper.Scrape(r.Context(), prob.URL); sErr == nil && len(scraped.TestCases) > 0 {
					req.TestCases = scraped.TestCases
					prob.TestCases = scraped.TestCases
					if prob.Statement == "" {
						prob.Statement = scraped.Statement
					}
					_ = h.problemRepo.Upsert(r.Context(), prob)
				}
			}
		}
	}

	if len(req.TestCases) == 0 {
		writeError(w, http.StatusBadRequest, "no test cases available for execution; provide test_cases in request or specify a problem_id with test cases")
		return
	}

	resp, err := h.judge0.ExecuteMultipleTestCases(r.Context(), req)
	if err != nil {
		slog.Error("judge0 batch execution failed", "error", err)
		writeError(w, http.StatusBadGateway, "batch execution failed: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

type scrapePayload struct {
	URL string `json:"url"`
}

// ScrapeProblem extracts problem statement, test cases, and starter templates from a CP platform URL.
// POST /api/v1/problems/scrape
func (h *PracticeHandler) ScrapeProblem(w http.ResponseWriter, r *http.Request) {
	var payload scrapePayload
	if !decodeJSON(w, r, &payload) {
		return
	}

	if strings.TrimSpace(payload.URL) == "" {
		writeError(w, http.StatusBadRequest, "url is required")
		return
	}

	if h.scraper == nil {
		writeError(w, http.StatusInternalServerError, "scraper is not configured")
		return
	}

	scraped, err := h.scraper.Scrape(r.Context(), payload.URL)
	if err != nil {
		slog.Error("problem scraping failed", "url", payload.URL, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to scrape problem: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, scraped)
}

