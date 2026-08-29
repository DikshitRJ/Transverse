// Package services contains domain business logic, psychometric models, and heuristic problem recommendation algorithms.
package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"transverse/internal/cache"
	"transverse/internal/config"
	"transverse/internal/models"
	"transverse/internal/repository"
)

// Judge0 execution status IDs.
const (
	Judge0Accepted            = 3
	Judge0WrongAnswer         = 4
	Judge0TimeLimitExceeded   = 5
	Judge0RuntimeError        = 6
	Judge0CompilationError    = 7
	Judge0MemoryLimitExceeded = 8
)

// PracticeService coordinates the adaptive practice session lifecycle,
// psychometric scoring, problem recommendation heuristics, and code execution evaluation.
type PracticeService struct {
	problemRepo  *repository.ProblemRepo
	statsRepo    *repository.StatsRepo
	sessionRepo  *repository.SessionRepo
	userRepo     *repository.UserRepo
	problemStats *repository.ProblemStatsRepo
	graphSvc     *GraphService
	cache        cache.Cache
	pool         *pgxpool.Pool
	cfg          *config.Config
	httpClient   *http.Client
}

// NewPracticeService constructs and initializes a new PracticeService instance.
func NewPracticeService(
	pool *pgxpool.Pool,
	problemRepo *repository.ProblemRepo,
	statsRepo *repository.StatsRepo,
	sessionRepo *repository.SessionRepo,
	userRepo *repository.UserRepo,
	problemStats *repository.ProblemStatsRepo,
	graphSvc *GraphService,
	cache cache.Cache,
	cfg *config.Config,
) *PracticeService {
	timeout := 5 * time.Second
	if cfg != nil && cfg.Judge0TimeoutMs > 0 {
		timeout = time.Duration(cfg.Judge0TimeoutMs) * time.Millisecond
	}

	return &PracticeService{
		pool:         pool,
		problemRepo:  problemRepo,
		statsRepo:    statsRepo,
		sessionRepo:  sessionRepo,
		userRepo:     userRepo,
		problemStats: problemStats,
		graphSvc:     graphSvc,
		cache:        cache,
		cfg:          cfg,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// StartRequest defines the parameters for initiating or resuming an adaptive practice session.
type StartRequest struct {
	UserID          string
	Mode            string   // "ADAPTIVE" | "REGULAR"
	Topics          []string
	Subtopics       []string
	Sources         []string
	DifficultyRange [2]int   // [min, max] Glicko rating; [0,0] means unrestricted
	Limit           int      // Max problems per session (0 = unlimited)
}

// StartResult encapsulates the initialized practice session and its initial problem recommendation.
type StartResult struct {
	Session      *models.PracticeSession
	FirstProblem *models.Problem
	ThetaStart   float64
}

// Start creates a new adaptive practice session or resumes an existing active session.
// Flow:
//  1. Resolve topic scope via GraphService
//  2. Check for existing ACTIVE session for same user (resume if found)
//  3. Load topic theta from topic_stats (fallback thetaDefault)
//  4. Load all problems in scope from DB
//  5. Load attempt counts ("seen:{userID}" cache)
//  6. Filter already-seen problems (ADAPTIVE: prefer unseen; REGULAR: hard exclude)
//  7. Cold-start: PickBestProblem(candidates, state, nil, false)
//  8. Create practice_sessions row (status=ACTIVE)
//  9. Return StartResult
func (s *PracticeService) Start(ctx context.Context, req StartRequest) (*StartResult, error) {
	if strings.TrimSpace(req.UserID) == "" {
		return nil, errors.New("practice_service: user_id cannot be empty")
	}

	// 1. Resolve topic scope via GraphService
	var resolvedTopics []string
	if len(req.Topics) > 0 && s.graphSvc != nil {
		res, err := s.graphSvc.ResolveScope(req.Topics)
		if err != nil {
			return nil, fmt.Errorf("practice_service: resolve topic scope: %w", err)
		}
		resolvedTopics = res
	} else {
		resolvedTopics = req.Topics
	}

	// 2. Check for existing ACTIVE session for same user (resume if found)
	activeSession, err := s.sessionRepo.GetActiveByUser(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("practice_service: check active session: %w", err)
	}
	if activeSession != nil {
		var currProblem *models.Problem
		if activeSession.CurrentProblemID != nil && *activeSession.CurrentProblemID != "" {
			currProblem, _ = s.problemRepo.GetByID(ctx, *activeSession.CurrentProblemID)
		}
		return &StartResult{
			Session:      activeSession,
			FirstProblem: currProblem,
			ThetaStart:   activeSession.ThetaStart,
		}, nil
	}

	// Ensure user profile exists
	user, err := s.userRepo.GetOrCreate(ctx, req.UserID, req.UserID, "")
	if err != nil {
		return nil, fmt.Errorf("practice_service: get or create user: %w", err)
	}

	// 3. Load topic theta from topic_stats (fallback to user.Theta or thetaDefault)
	thetaStart := thetaDefault
	if len(resolvedTopics) > 0 {
		topicStat, err := s.statsRepo.GetByUserAndTopic(ctx, req.UserID, resolvedTopics[0])
		if err == nil && topicStat != nil && topicStat.Theta > 0 {
			thetaStart = topicStat.Theta
		}
	} else if user.Theta > 0 {
		thetaStart = user.Theta
	}

	// 4. Load all problems in scope from DB
	minRating := float64(req.DifficultyRange[0])
	maxRating := float64(req.DifficultyRange[1])
	scopeProblems, err := s.problemRepo.GetByScope(ctx, models.SessionScope{Topics: resolvedTopics, Sources: req.Sources, DifficultyRange: [2]int{int(minRating), int(maxRating)}})
	if err != nil {
		return nil, fmt.Errorf("practice_service: load scoped problems: %w", err)
	}
	if len(scopeProblems) == 0 {
		if len(resolvedTopics) > 0 {
			scopeProblems, err = s.problemRepo.GetByScope(ctx, models.SessionScope{Topics: resolvedTopics})
			if err != nil {
				return nil, fmt.Errorf("practice_service: fallback load topic problems: %w", err)
			}
		}
		if len(scopeProblems) == 0 {
			return nil, errors.New("practice_service: no problems found matching requested scope")
		}
	}

	// 5. Load attempt counts ("seen:{userID}" cache)
	attemptCounts, err := s.loadSeenIDs(ctx, req.UserID)
	if err != nil {
		attemptCounts = make(map[string]int)
	}

	// 6. Filter already-seen problems
	mode := strings.ToUpper(strings.TrimSpace(req.Mode))
	if mode == "" {
		mode = "ADAPTIVE"
	}

	var candidates []models.Problem
	if mode == "REGULAR" {
		for _, p := range scopeProblems {
			if attemptCounts[p.ID] == 0 {
				candidates = append(candidates, p)
			}
		}
		if len(candidates) == 0 {
			candidates = scopeProblems
		}
	} else {
		var unseen []models.Problem
		for _, p := range scopeProblems {
			if attemptCounts[p.ID] == 0 {
				unseen = append(unseen, p)
			}
		}
		if len(unseen) > 0 {
			candidates = unseen
		} else {
			candidates = scopeProblems
		}
	}

	// 7. Cold-start: PickBestProblem
	scope := models.SessionScope{
		Topics:          resolvedTopics,
		Subtopics:       req.Subtopics,
		Sources:         req.Sources,
		DifficultyRange: req.DifficultyRange,
	}
	state := s.buildScState(ctx, req.UserID, thetaStart, scope, nil)
	state.ThetaCurrent = thetaStart

	pickRes := PickBestProblem(candidates, state, nil, false)
	var firstProblem *models.Problem
	if pickRes != nil && pickRes.Problem != nil {
		firstProblem = pickRes.Problem
	} else if len(candidates) > 0 {
		firstProblem = &candidates[0]
	} else {
		return nil, errors.New("practice_service: failed to select initial problem candidate")
	}

	// 8. Create practice_sessions row (status=ACTIVE)
	scopeBytes, err := json.Marshal(scope)
	if err != nil {
		scopeBytes = []byte("{}")
	}

	sessionID := generateID("sess")
	now := time.Now()
	session := &models.PracticeSession{
		ID:               sessionID,
		UserID:           req.UserID,
		Mode:             mode,
		ScopeRaw:         scopeBytes,
		ThetaStart:       thetaStart,
		ThetaCurrent:     thetaStart,
		CurrentProblemID: &firstProblem.ID,
		ResponsesRaw:     []byte("[]"),
		QuestionCount:    0,
		Status:           "ACTIVE",
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("practice_service: create practice session: %w", err)
	}

	return &StartResult{
		Session:      session,
		FirstProblem: firstProblem,
		ThetaStart:   thetaStart,
	}, nil
}

// SubmitRequest defines the payload for submitting a problem execution verdict to the session engine.
type SubmitRequest struct {
	UserID      string
	SessionID   string
	ProblemID   string
	Judge0Token string
	TimeTakenMs int64
}

// VerdictDetail contains detailed outcome and execution metrics from Judge0.
type VerdictDetail struct {
	StatusID   int    `json:"status_id"`
	StatusDesc string `json:"status_desc"`
	TimeMs     int    `json:"time_ms"`
	MemoryKB   int    `json:"memory_kb"`
	Stderr     string `json:"stderr,omitempty"`
	CompileOut string `json:"compile_out,omitempty"`
}

// SubmitResult summarizes the evaluation outcome, psychometric shift, and next problem recommendation.
type SubmitResult struct {
	IsCorrect   bool
	Verdict     VerdictDetail
	ThetaBefore float64
	ThetaAfter  float64
	NextProblem *models.Problem
}

type judge0APIResponse struct {
	Status struct {
		ID          int    `json:"id"`
		Description string `json:"description"`
	} `json:"status"`
	Time          interface{} `json:"time"`
	Memory        interface{} `json:"memory"`
	Stderr        *string     `json:"stderr"`
	CompileOutput *string     `json:"compile_output"`
	Message       *string     `json:"message"`
}

// fetchJudge0Verdict fetches execution verdict details for the given Judge0 tracking token.
func (s *PracticeService) fetchJudge0Verdict(ctx context.Context, token string) (VerdictDetail, error) {
	if strings.TrimSpace(token) == "" {
		return VerdictDetail{}, errors.New("practice_service: judge0 token cannot be empty")
	}

	baseURL := "https://judge0-ce.p.rapidapi.com"
	apiKey := ""
	if s.cfg != nil {
		if s.cfg.Judge0BaseURL != "" {
			baseURL = strings.TrimRight(s.cfg.Judge0BaseURL, "/")
		}
		apiKey = s.cfg.Judge0APIKey
	}

	endpoint := fmt.Sprintf("%s/submissions/%s?base64_encoded=false&fields=status,time,memory,stderr,compile_output,message", baseURL, url.PathEscape(token))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return VerdictDetail{}, fmt.Errorf("practice_service: create judge0 request: %w", err)
	}

	if apiKey != "" {
		req.Header.Set("X-RapidAPI-Key", apiKey)
		req.Header.Set("X-Auth-Token", apiKey)
		if u, parseErr := url.Parse(baseURL); parseErr == nil && u.Host != "" {
			req.Header.Set("X-RapidAPI-Host", u.Host)
		}
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return VerdictDetail{}, fmt.Errorf("practice_service: execute judge0 request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return VerdictDetail{}, fmt.Errorf("practice_service: judge0 returned non-200 status %d: %s", resp.StatusCode, string(body))
	}

	var raw judge0APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return VerdictDetail{}, fmt.Errorf("practice_service: decode judge0 response: %w", err)
	}

	var timeMs int
	switch v := raw.Time.(type) {
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			timeMs = int(math.Round(f * 1000.0))
		}
	case float64:
		timeMs = int(math.Round(v * 1000.0))
	}

	var memoryKB int
	switch v := raw.Memory.(type) {
	case float64:
		memoryKB = int(v)
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			memoryKB = i
		}
	}

	stderr := ""
	if raw.Stderr != nil {
		stderr = *raw.Stderr
	}

	compileOut := ""
	if raw.CompileOutput != nil {
		compileOut = *raw.CompileOutput
	}

	return VerdictDetail{
		StatusID:   raw.Status.ID,
		StatusDesc: raw.Status.Description,
		TimeMs:     timeMs,
		MemoryKB:   memoryKB,
		Stderr:     stderr,
		CompileOut: compileOut,
	}, nil
}

// Submit processes a Judge0 submission verdict, updates latent ability (IRT theta),
// records submission telemetry, and selects the next problem recommendation.
func (s *PracticeService) Submit(ctx context.Context, req SubmitRequest) (*SubmitResult, error) {
	if req.SessionID == "" || req.ProblemID == "" {
		return nil, errors.New("practice_service: session_id and problem_id are required")
	}

	// 1. Validate session + ownership + ACTIVE status
	session, err := s.sessionRepo.GetByID(ctx, req.SessionID)
	if err != nil {
		return nil, fmt.Errorf("practice_service: get session: %w", err)
	}
	if session == nil {
		return nil, errors.New("practice_service: session not found")
	}
	if session.UserID != req.UserID {
		return nil, errors.New("practice_service: unauthorized - session belongs to another user")
	}
	if session.Status != "ACTIVE" {
		return nil, fmt.Errorf("practice_service: session is not active (status=%s)", session.Status)
	}

	// 2. Load problem from cache/DB
	problem, err := s.problemRepo.GetByID(ctx, req.ProblemID)
	if err != nil {
		return nil, fmt.Errorf("practice_service: load problem %q: %w", req.ProblemID, err)
	}
	if problem == nil {
		return nil, fmt.Errorf("practice_service: problem %q not found", req.ProblemID)
	}

	// 3. Fetch verdict from Judge0
	verdict, err := s.fetchJudge0Verdict(ctx, req.Judge0Token)
	if err != nil {
		return nil, fmt.Errorf("practice_service: fetch judge0 verdict: %w", err)
	}

	// 4. If Compilation Error (status 7): do NOT update theta, return same problem
	if verdict.StatusID == Judge0CompilationError {
		return &SubmitResult{
			IsCorrect:   false,
			Verdict:     verdict,
			ThetaBefore: session.ThetaCurrent,
			ThetaAfter:  session.ThetaCurrent,
			NextProblem: problem,
		}, nil
	}

	// 5. Determine isCorrect
	isCorrect := (verdict.StatusID == Judge0Accepted)

	// 6. IRT theta update
	thetaBefore := session.ThetaCurrent
	thetaAfter := ComputeThetaUpdate(thetaBefore, problem.GlickoRating, isCorrect, req.TimeTakenMs, int64(problem.AvgTimeMs))

	// 7. Compute streaks from response history
	pastResponses, err := session.Responses()
	if err != nil {
		pastResponses = []models.SessionResponse{}
	}
	consecCorrect, consecWrong := ComputeStreaks(pastResponses, isCorrect)

	// 8. Load DNA
	dna, _ := s.loadDNA(ctx, req.UserID)

	// 9. Build ScState
	scope, _ := session.Scope()
	state := s.buildScState(ctx, req.UserID, thetaAfter, scope, pastResponses)
	state.ConsecutiveCorrect = consecCorrect
	state.ConsecutiveWrong = consecWrong
	state.QuestionCount = session.QuestionCount + 1
	state.AvgSessionLength = dna.AvgSessionLength
	state.AvgTimeTakenMs = float64(dna.AvgTimeTakenMs)
	state.CarelessnessIndex = dna.CarelessnessIndex

	// 10. Pick next problem
	sessionSeen := make(map[string]bool, len(pastResponses)+1)
	for _, r := range pastResponses {
		sessionSeen[r.ProblemID] = true
	}
	sessionSeen[problem.ID] = true

	var candidates []models.Problem
	if session.Mode == "ADAPTIVE" && !isCorrect && len(problem.Embedding.Slice()) > 0 {
		similar, err := s.problemRepo.FindSimilar(ctx, problem.Embedding, problem.Topic, 30)
		if err == nil && len(similar) > 0 {
			for _, p := range similar {
				if !sessionSeen[p.ID] {
					candidates = append(candidates, p)
				}
			}
		}
	}

	if len(candidates) == 0 {
		scoped, err := s.problemRepo.GetByScope(ctx, scope)
		if err == nil {
			for _, p := range scoped {
				if !sessionSeen[p.ID] {
					candidates = append(candidates, p)
				}
			}
		}
	}

	if len(candidates) == 0 {
		scoped, _ := s.problemRepo.GetByScope(ctx, scope)
		for _, p := range scoped {
			if p.ID != problem.ID {
				candidates = append(candidates, p)
			}
		}
	}

	if len(candidates) == 0 && len(scope.Topics) > 0 {
		topicProbs, _ := s.problemRepo.GetByScope(ctx, models.SessionScope{Topics: scope.Topics})
		for _, p := range topicProbs {
			if p.ID != problem.ID {
				candidates = append(candidates, p)
			}
		}
	}

	var nextProblem *models.Problem
	var pickRes *PickResult
	if len(candidates) > 0 {
		pickRes = PickBestProblem(candidates, state, problem, isCorrect)
		if pickRes != nil {
			nextProblem = pickRes.Problem
		}
	}

	// 11. Build SessionResponse
	now := time.Now()
	var scScore, df, cs, tp, nf, ir, pd, cp, thetaEff, momentum float64
	if pickRes != nil {
		scScore = pickRes.Scores.Total
		df = pickRes.Scores.DifficultyFit
		cs = pickRes.Scores.ConceptSimilarity
		tp = pickRes.Scores.TopicProgression
		nf = pickRes.Scores.NoveltyFactor
		ir = pickRes.Scores.ImmediateReinforce
		pd = pickRes.Scores.PlatformDiversity
		cp = pickRes.Scores.CarelessnessPenalty
		thetaEff = pickRes.ThetaEff
		momentum = pickRes.Momentum
	}

	respRecord := models.SessionResponse{
		ProblemID:           problem.ID,
		IsCorrect:           isCorrect,
		Skipped:             false,
		Judge0StatusID:      verdict.StatusID,
		Judge0StatusDesc:    verdict.StatusDesc,
		ExecutionTimeMs:     verdict.TimeMs,
		MemoryKB:            verdict.MemoryKB,
		TimeTakenMs:         req.TimeTakenMs,
		ThetaBefore:         thetaBefore,
		ThetaAfter:          thetaAfter,
		QuestionCount:       session.QuestionCount + 1,
		ScScore:             scScore,
		DifficultyFit:       df,
		ConceptSimilarity:   cs,
		TopicProgression:    tp,
		NoveltyFactor:       nf,
		ImmediateReinforce:  ir,
		PlatformDiversity:   pd,
		CarelessnessPenalty: cp,
		ThetaEffective:      thetaEff,
		Momentum:            momentum,
		SubmittedAt:         now,
	}

	// 12. Submit operations
	if err := s.problemStats.RecordAttempt(ctx, req.UserID, req.ProblemID, isCorrect, req.TimeTakenMs); err != nil {
		return nil, fmt.Errorf("practice_service: record problem stats: %w", err)
	}

	var nextProblemID *string
	if nextProblem != nil {
		nextProblemID = &nextProblem.ID
	}

	if err := s.sessionRepo.AppendResponse(ctx, session.ID, respRecord, nextProblemID, thetaAfter); err != nil {
		return nil, fmt.Errorf("practice_service: append response in session: %w", err)
	}

	// 13. Invalidate "seen:{userID}" cache
	if s.cache != nil {
		_ = s.cache.Del(ctx, CacheKeySeenIDs(req.UserID))
	}

	// 14. UpdateStats on problem (async, best-effort)
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := s.problemRepo.IncrementAttemptCount(bgCtx, req.ProblemID, isCorrect); err != nil {
			slog.Warn("practice_service: failed to update problem stats", "problem_id", req.ProblemID, "error", err)
		}
	}()

	// 15. Return SubmitResult
	return &SubmitResult{
		IsCorrect:   isCorrect,
		Verdict:     verdict,
		ThetaBefore: thetaBefore,
		ThetaAfter:  thetaAfter,
		NextProblem: nextProblem,
	}, nil
}

// SkipResult represents the response payload for a skipped problem.
type SkipResult struct {
	ThetaBefore float64
	ThetaAfter  float64
	NextProblem *models.Problem
}

// Skip advances past the current problem without altering ability (theta),
// resets consecutive streaks, records a skip telemetry record, and selects the next problem.
func (s *PracticeService) Skip(ctx context.Context, userID, sessionID, problemID string, timeTakenMs int64) (*SkipResult, error) {
	if sessionID == "" || problemID == "" {
		return nil, errors.New("practice_service: session_id and problem_id are required")
	}

	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("practice_service: get session: %w", err)
	}
	if session == nil {
		return nil, errors.New("practice_service: session not found")
	}
	if session.UserID != userID {
		return nil, errors.New("practice_service: unauthorized - session belongs to another user")
	}
	if session.Status != "ACTIVE" {
		return nil, fmt.Errorf("practice_service: session is not active (status=%s)", session.Status)
	}

	problem, err := s.problemRepo.GetByID(ctx, problemID)
	if err != nil {
		return nil, fmt.Errorf("practice_service: load problem %q: %w", problemID, err)
	}
	if problem == nil {
		return nil, fmt.Errorf("practice_service: problem %q not found", problemID)
	}

	thetaBefore := session.ThetaCurrent
	thetaAfter := session.ThetaCurrent // No theta change on skip

	pastResponses, err := session.Responses()
	if err != nil {
		pastResponses = []models.SessionResponse{}
	}

	scope, _ := session.Scope()
	state := s.buildScState(ctx, userID, thetaAfter, scope, pastResponses)
	state.ConsecutiveCorrect = 0
	state.ConsecutiveWrong = 0
	state.QuestionCount = session.QuestionCount + 1

	sessionSeen := make(map[string]bool, len(pastResponses)+1)
	for _, r := range pastResponses {
		sessionSeen[r.ProblemID] = true
	}
	sessionSeen[problem.ID] = true

	var candidates []models.Problem
	scoped, err := s.problemRepo.GetByScope(ctx, scope)
	if err == nil {
		for _, p := range scoped {
			if !sessionSeen[p.ID] {
				candidates = append(candidates, p)
			}
		}
	}
	if len(candidates) == 0 {
		for _, p := range scoped {
			if p.ID != problem.ID {
				candidates = append(candidates, p)
			}
		}
	}

	var nextProblem *models.Problem
	var pickRes *PickResult
	if len(candidates) > 0 {
		pickRes = PickAfterSkip(candidates, state, problem)
		if pickRes != nil {
			nextProblem = pickRes.Problem
		}
	}

	now := time.Now()
	var scScore, df, cs, tp, nf, ir, pd, cp, thetaEff, momentum float64
	if pickRes != nil {
		scScore = pickRes.Scores.Total
		df = pickRes.Scores.DifficultyFit
		cs = pickRes.Scores.ConceptSimilarity
		tp = pickRes.Scores.TopicProgression
		nf = pickRes.Scores.NoveltyFactor
		ir = pickRes.Scores.ImmediateReinforce
		pd = pickRes.Scores.PlatformDiversity
		cp = pickRes.Scores.CarelessnessPenalty
		thetaEff = pickRes.ThetaEff
		momentum = pickRes.Momentum
	}

	respRecord := models.SessionResponse{
		ProblemID:           problem.ID,
		IsCorrect:           false,
		Skipped:             true,
		Judge0StatusID:      0,
		Judge0StatusDesc:    "SKIPPED",
		ExecutionTimeMs:     0,
		MemoryKB:            0,
		TimeTakenMs:         timeTakenMs,
		ThetaBefore:         thetaBefore,
		ThetaAfter:          thetaAfter,
		QuestionCount:       session.QuestionCount + 1,
		ScScore:             scScore,
		DifficultyFit:       df,
		ConceptSimilarity:   cs,
		TopicProgression:    tp,
		NoveltyFactor:       nf,
		ImmediateReinforce:  ir,
		PlatformDiversity:   pd,
		CarelessnessPenalty: cp,
		ThetaEffective:      thetaEff,
		Momentum:            momentum,
		SubmittedAt:         now,
	}

	if err := s.problemStats.RecordAttempt(ctx, userID, problemID, false, timeTakenMs); err != nil {
		return nil, fmt.Errorf("practice_service: record problem stats: %w", err)
	}

	var nextProblemID *string
	if nextProblem != nil {
		nextProblemID = &nextProblem.ID
	}

	if err := s.sessionRepo.AppendResponse(ctx, session.ID, respRecord, nextProblemID, thetaAfter); err != nil {
		return nil, fmt.Errorf("practice_service: append response in session: %w", err)
	}

	if s.cache != nil {
		_ = s.cache.Del(ctx, CacheKeySeenIDs(userID))
	}

	return &SkipResult{
		ThetaBefore: thetaBefore,
		ThetaAfter:  thetaAfter,
		NextProblem: nextProblem,
	}, nil
}

// CloseResult summarizes the final outcomes, mastery ratings, and topic breakdowns upon session completion.
type CloseResult struct {
	SessionID      string                 `json:"session_id"`
	ThetaStart     float64                `json:"theta_start"`
	ThetaFinal     float64                `json:"theta_final"`
	MasteryScore   float64                `json:"mastery_score"`
	TotalProblems  int                    `json:"total_problems"`
	CorrectCount   int                    `json:"correct_count"`
	Accuracy       float64                `json:"accuracy"`
	AvgTimeMs      int64                  `json:"avg_time_ms"`
	TopicBreakdown map[string]TopicResult `json:"topic_breakdown"`
}

// TopicResult summarizes user performance in a single topic during the practice session.
type TopicResult struct {
	Topic        string  `json:"topic"`
	Correct      int     `json:"correct"`
	Total        int     `json:"total"`
	Accuracy     float64 `json:"accuracy"`
	ThetaChange  float64 `json:"theta_change"`
	MasteryScore float64 `json:"mastery_score"`
}

// Close finalizes an active practice session, evaluates Glicko-2 ratings,
// updates per-topic mastery metrics, and recomputes the user's Learning DNA.
func (s *PracticeService) Close(ctx context.Context, userID, sessionID string) (*CloseResult, error) {
	if sessionID == "" {
		return nil, errors.New("practice_service: session_id is required")
	}

	// 1. Validate session + ACTIVE
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("practice_service: get session: %w", err)
	}
	if session == nil {
		return nil, errors.New("practice_service: session not found")
	}
	if session.UserID != userID {
		return nil, errors.New("practice_service: unauthorized - session belongs to another user")
	}
	if session.Status != "ACTIVE" {
		return nil, fmt.Errorf("practice_service: session is not active (status=%s)", session.Status)
	}

	// 2. If no responses: Abandon session and return baseline result
	responses, err := session.Responses()
	if err != nil || len(responses) == 0 {
		_ = s.sessionRepo.UpdateStatus(ctx, sessionID, "ABANDONED")
		return &CloseResult{
			SessionID:      sessionID,
			ThetaStart:     session.ThetaStart,
			ThetaFinal:     session.ThetaCurrent,
			MasteryScore:   ComputeMasteryScore(session.ThetaCurrent),
			TotalProblems:  0,
			CorrectCount:   0,
			Accuracy:       0.0,
			AvgTimeMs:      0,
			TopicBreakdown: make(map[string]TopicResult),
		}, nil
	}

	// 3. Compute session performance statistics
	totalProblems := len(responses)
	correctCount := 0
	var totalTimeMs int64

	problemIDs := make([]string, 0, totalProblems)
	for _, r := range responses {
		problemIDs = append(problemIDs, r.ProblemID)
	}

	problems, err := s.problemRepo.GetByIDs(ctx, problemIDs)
	if err != nil {
		return nil, fmt.Errorf("practice_service: batch get problems: %w", err)
	}

	problemMap := make(map[string]models.Problem, len(problems))
	for _, p := range problems {
		problemMap[p.ID] = p
	}

	topicCorrect := make(map[string]int)
	topicTotal := make(map[string]int)

	for _, r := range responses {
		if r.IsCorrect {
			correctCount++
		}
		totalTimeMs += r.TimeTakenMs
		if p, ok := problemMap[r.ProblemID]; ok && p.Topic != "" {
			topicTotal[p.Topic]++
			if r.IsCorrect {
				topicCorrect[p.Topic]++
			}
		}
	}

	accuracy := float64(correctCount) / float64(totalProblems)
	var avgTimeMs int64
	if totalProblems > 0 {
		avgTimeMs = totalTimeMs / int64(totalProblems)
	}

	// 4. Load user and DNA
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return nil, fmt.Errorf("practice_service: load user %q: %w", userID, err)
	}
	dna, _ := user.DNA()

	// 5. Compute average opponent rating
	avgOpponentRating := ComputeAvgOpponentRating(responses, problemMap)

	// 6. Glicko-2 psychometric update
	glickoIn := GlickoSessionInput{
		PlayerRating:      user.GlickoRating,
		PlayerRD:          user.GlickoRD,
		PlayerVol:         user.GlickoVol,
		AvgOpponentRating: avgOpponentRating,
		Score:             accuracy,
	}
	glickoOut := UpdateGlickoFromSession(glickoIn)

	// 7. Mastery score computation
	thetaFinal := session.ThetaCurrent
	masteryScore := ComputeMasteryScore(thetaFinal)

	// 8. Recompute Learning DNA
	sessionDurationMs := session.UpdatedAt.Sub(session.CreatedAt).Milliseconds()
	if sessionDurationMs <= 0 {
		sessionDurationMs = totalTimeMs
	}
	newDNA := UpdateLearningDNA(dna, responses, time.Duration(sessionDurationMs) * time.Millisecond)

	// Compute Topic Breakdown and per-topic updates
	topicBreakdown := make(map[string]TopicResult, len(topicTotal))
	topicStatsToUpsert := make([]models.TopicStats, 0, len(topicTotal))

	thetaDelta := thetaFinal - session.ThetaStart

	for topic, total := range topicTotal {
		correct := topicCorrect[topic]
		topicAcc := float64(correct) / float64(total)

		existingTopicStat, _ := s.statsRepo.GetByUserAndTopic(ctx, userID, topic)
		existingTopicTheta := thetaDefault
		existingAttemptCount := 0
		existingCorrectCount := 0
		existingGlickoRating := 1500.0

		if existingTopicStat != nil {
			existingTopicTheta = existingTopicStat.Theta
			existingAttemptCount = existingTopicStat.AttemptCount
			existingCorrectCount = existingTopicStat.CorrectCount
			existingGlickoRating = existingTopicStat.GlickoRating
		}

		newTopicTheta := clamp(existingTopicTheta+thetaDelta, thetaFloor, thetaCeiling)
		newTopicMastery := ComputeMasteryScore(newTopicTheta)

		topicBreakdown[topic] = TopicResult{
			Topic:        topic,
			Correct:      correct,
			Total:        total,
			Accuracy:     topicAcc,
			ThetaChange:  thetaDelta,
			MasteryScore: newTopicMastery,
		}

		topicStatsToUpsert = append(topicStatsToUpsert, models.TopicStats{
			UserID:       userID,
			Topic:        topic,
			Theta:        newTopicTheta,
			MasteryScore: newTopicMastery,
			GlickoRating: existingGlickoRating,
			AttemptCount: existingAttemptCount + total,
			CorrectCount: existingCorrectCount + correct,
		})
	}

	// 9. Update records without full TX as we refactored
	dnaBytes, err := json.Marshal(newDNA)
	if err != nil {
		dnaBytes = []byte("{}")
	}

	userUpdateQuery := `
		UPDATE users SET
			theta = $1,
			glicko_rating = $2,
			glicko_rd = $3,
			glicko_vol = $4,
			dna = $5,
			updated_at = NOW()
		WHERE id = $6
	`
	if _, err := s.pool.Exec(ctx, userUpdateQuery, thetaFinal, glickoOut.NewRating, glickoOut.NewRD, glickoOut.NewVol, dnaBytes, userID); err != nil {
		return nil, fmt.Errorf("practice_service: update user psychometrics: %w", err)
	}

	for _, ts := range topicStatsToUpsert {
		if err := s.statsRepo.Upsert(ctx, &ts); err != nil {
			return nil, fmt.Errorf("practice_service: upsert topic stats for %q: %w", ts.Topic, err)
		}
	}

	if err := s.sessionRepo.CloseSession(ctx, sessionID, thetaFinal); err != nil {
		return nil, fmt.Errorf("practice_service: close session: %w", err)
	}

	// Invalidate user, DNA, and topic caches
	if s.cache != nil {
		_ = s.cache.Del(ctx, fmt.Sprintf("user:%s", userID))
		_ = s.cache.Del(ctx, CacheKeyDNA(userID))
		_ = s.cache.Del(ctx, CacheKeyTopicStats(userID))
	}

	// 10. Return CloseResult
	return &CloseResult{
		SessionID:      sessionID,
		ThetaStart:     session.ThetaStart,
		ThetaFinal:     thetaFinal,
		MasteryScore:   masteryScore,
		TotalProblems:  totalProblems,
		CorrectCount:   correctCount,
		Accuracy:       accuracy,
		AvgTimeMs:      avgTimeMs,
		TopicBreakdown: topicBreakdown,
	}, nil
}

// GetSession returns the complete state of a session and its current problem (used for resumption).
func (s *PracticeService) GetSession(ctx context.Context, userID, sessionID string) (*models.PracticeSession, *models.Problem, error) {
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, nil, fmt.Errorf("practice_service: get session %q: %w", sessionID, err)
	}
	if session == nil {
		return nil, nil, errors.New("practice_service: session not found")
	}
	if session.UserID != userID {
		return nil, nil, errors.New("practice_service: unauthorized - session belongs to another user")
	}

	var currentProblem *models.Problem
	if session.CurrentProblemID != nil && *session.CurrentProblemID != "" {
		currentProblem, _ = s.problemRepo.GetByID(ctx, *session.CurrentProblemID)
	}

	return session, currentProblem, nil
}

// GetSimilar returns top-k similar problems to a given problem using pgvector ANN.
func (s *PracticeService) GetSimilar(ctx context.Context, problemID, topic string, limit int) ([]models.Problem, error) {
	problem, err := s.problemRepo.GetByID(ctx, problemID)
	if err != nil {
		return nil, fmt.Errorf("practice_service: get problem %q: %w", problemID, err)
	}
	if problem == nil {
		return nil, fmt.Errorf("practice_service: problem %q not found", problemID)
	}

	if len(problem.Embedding.Slice()) == 0 {
		return nil, errors.New("practice_service: problem has no embedding vector")
	}

	targetTopic := topic
	if targetTopic == "" {
		targetTopic = problem.Topic
	}

	return s.problemRepo.FindSimilar(ctx, problem.Embedding, targetTopic, limit)
}

// GetTopics returns all topic statistics with current mastery scores for a user.
func (s *PracticeService) GetTopics(ctx context.Context, userID string) ([]models.TopicStats, error) {
	return s.statsRepo.GetByUser(ctx, userID)
}

func (s *PracticeService) loadSeenIDs(ctx context.Context, userID string) (map[string]int, error) {
	return s.problemStats.GetAttemptCountsByUser(ctx, userID)
}

func (s *PracticeService) loadDNA(ctx context.Context, userID string) (models.LearningDNA, error) {
	if s.cache != nil {
		var cached models.LearningDNA
		if err := s.cache.Get(ctx, CacheKeyDNA(userID), &cached); err == nil {
			return cached, nil
		}
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return models.DefaultDNA(), fmt.Errorf("practice_service: load dna user lookup: %w", err)
	}
	if user == nil {
		return models.DefaultDNA(), nil
	}

	dna, err := user.DNA()
	if err != nil {
		return models.DefaultDNA(), nil
	}

	if s.cache != nil {
		_ = s.cache.Set(ctx, CacheKeyDNA(userID), dna, 60*time.Second)
	}

	return dna, nil
}

func (s *PracticeService) buildScState(
	ctx context.Context,
	userID string,
	thetaCurrent float64,
	scope models.SessionScope,
	responses []models.SessionResponse,
) *ScState {
	dna, err := s.loadDNA(ctx, userID)
	if err != nil {
		dna = models.DefaultDNA()
	}

	attemptCounts, err := s.loadSeenIDs(ctx, userID)
	if err != nil {
		attemptCounts = make(map[string]int)
	}

	primaryTopic := ""
	if len(scope.Topics) > 0 {
		primaryTopic = scope.Topics[0]
	}

	sessionSourceCounts := make(map[string]int)
	if len(responses) > 0 {
		pIDs := make([]string, 0, len(responses))
		for _, r := range responses {
			pIDs = append(pIDs, r.ProblemID)
		}
		problems, err := s.problemRepo.GetByIDs(ctx, pIDs)
		if err == nil {
			for _, p := range problems {
				if p.Source != "" {
					sessionSourceCounts[p.Source]++
				}
			}
		}
	}

	return &ScState{
		ThetaCurrent:        thetaCurrent,
		Topic:               primaryTopic,
		TopicBias:           dna.TopicBias[primaryTopic],
		ConsecutiveCorrect:  0,
		ConsecutiveWrong:    0,
		QuestionCount:       len(responses),
		AvgSessionLength:    dna.AvgSessionLength,
		AvgTimeTakenMs:      float64(dna.AvgTimeTakenMs),
		CarelessnessIndex:   dna.CarelessnessIndex,
		AttemptCounts:       attemptCounts,
		ActiveSources:       scope.Sources,
		SessionSourceCounts: sessionSourceCounts,
	}
}

// SearchProblems executes filtered search across problem names, tags, sources, topics, and difficulties.
func (s *PracticeService) SearchProblems(ctx context.Context, req models.ProblemSearchRequest) (*models.ProblemSearchResponse, error) {
	problems, total, err := s.problemRepo.Search(ctx, req)
	if err != nil {
		return nil, err
	}
	return &models.ProblemSearchResponse{
		Problems: models.ToProblemPayloads(problems),
		Total:    total,
	}, nil
}
