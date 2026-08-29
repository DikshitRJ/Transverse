package services

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"math/rand/v2"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"

	"velocity/internal/cache"
	"velocity/internal/graph"
	"velocity/internal/models"
	"velocity/internal/repository"
)

type LearnService struct {
	questionRepo  *repository.QuestionRepo
	statsRepo     *repository.StatsRepo
	sessionRepo   *repository.SessionRepo
	userRepo      *repository.UserRepo
	questionStats *repository.QuestionStatsRepo
	syllabusGraph graph.SyllabusGraph
	cache         cache.Cache
	pool          *pgxpool.Pool
}

func NewLearnService(
	qr *repository.QuestionRepo,
	sr *repository.StatsRepo,
	ssr *repository.SessionRepo,
	ur *repository.UserRepo,
	qsr *repository.QuestionStatsRepo,
	sg graph.SyllabusGraph,
	c cache.Cache,
	pool *pgxpool.Pool,
) *LearnService {
	return &LearnService{
		questionRepo:  qr,
		statsRepo:     sr,
		sessionRepo:   ssr,
		userRepo:      ur,
		questionStats: qsr,
		syllabusGraph: sg,
		cache:         c,
		pool:          pool,
	}
}

type StartResult struct {
	Session             *models.LearnSession
	FirstQuestion       *models.Question
	ThetaStart          float64
	QuestionMap         []models.QuestionMapItem
	TotalScopeQuestions int // total questions matching the scope (before any limit)
}

func (s *LearnService) SyllabusGraph() graph.SyllabusGraph {
	return s.syllabusGraph
}

func (s *LearnService) Start(ctx context.Context, userID, mode string, chapters, chapterGroups, subjects, years, examTypes []string, questionLimit int, biometricEnabled bool, questionOrdering string) (*StartResult, error) {
	resolved, err := s.questionRepo.ResolveScopeByDB(ctx, subjects, chapterGroups, chapters, years, examTypes)
	if err != nil {
		return nil, fmt.Errorf("learn: resolve scope: %w", err)
	}
	if len(resolved) == 0 {
		return nil, fmt.Errorf("learn: no chapters matched the given scope")
	}
	primaryChapter := resolved[0]

	if len(resolved) == 1 {
		existing, err := s.sessionRepo.GetActiveByUserAndChapter(ctx, userID, primaryChapter)
		if err == nil && existing != nil {
			q, err := s.loadCurrentQuestion(ctx, existing)
			if err == nil {
				return &StartResult{
					Session:       existing,
					FirstQuestion: q,
					ThetaStart:    float64(existing.ThetaCurrent),
				}, nil
			}
			_ = s.sessionRepo.Abandon(ctx, existing.ID)
		}
	}

	thetaStart := 1300.0
	cs, err := s.statsRepo.GetChapterStats(ctx, userID, primaryChapter)
	if err == nil && cs != nil {
		thetaStart = cs.Theta
	}

	candidates, err := s.loadQuestionsFiltered(ctx, resolved, years, examTypes)
	if err != nil {
		return nil, fmt.Errorf("learn: load questions: %w", err)
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("learn: no questions in the given scope")
	}

	seenIDs, err := s.loadSeenIDs(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("learn: load seen: %w", err)
	}

	ordering := questionOrdering
	if ordering == "" {
		ordering = "random"
	}

	if questionLimit <= 0 || questionLimit > len(candidates) {
		questionLimit = len(candidates)
	}

	// For REGULAR mode: sort by ordering and trim to question limit.
	// The session only covers the first N questions in the chosen order.
	var orderedScope []models.Question
	if mode == "REGULAR" {
		orderedScope = sortQuestionsByOrdering(candidates, ordering)
		if len(orderedScope) > questionLimit {
			orderedScope = orderedScope[:questionLimit]
		}
		questionLimit = len(orderedScope)
	}

	var firstQ *models.Question
	if mode == "REGULAR" {
		unseen := filterUnseen(orderedScope, seenIDs)
		if len(unseen) == 0 {
			return nil, fmt.Errorf("learn: all questions in the limited scope have been attempted")
		}
		firstQ = &unseen[0]
	} else {
		unseen := filterUnseen(candidates, seenIDs)
		if len(unseen) == 0 {
			unseen = candidates
		}
		state := s.buildScState(ctx, userID, thetaStart, "")
		state.AttemptCounts = seenIDs
		result := PickBestQuestion(unseen, state, nil, false)
		firstQ = result.Question
	}

	sessionID := generateID()
	scope := models.SessionScope{
		Chapters:      resolved,
		ChapterGroups: chapterGroups,
		Subjects:      subjects,
		Years:         years,
		ExamTypes:     examTypes,
	}
	scopeRaw, _ := json.Marshal(scope)

	session := &models.LearnSession{
		ID:                sessionID,
		UserID:            userID,
		Mode:              mode,
		Chapter:           primaryChapter,
		ScopeRaw:          scopeRaw,
		ThetaStart:        float32(thetaStart),
		ThetaCurrent:      float32(thetaStart),
		QuestionCount:     0,
		CurrentQuestionID: &firstQ.ID,
		Status:            "ACTIVE",
		QuestionLimit:     questionLimit,
		BiometricEnabled:  biometricEnabled,
		QuestionOrdering:  ordering,
	}

	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("learn: create session: %w", err)
	}

	// Build question_map for REGULAR mode — trimmed to questionLimit, ordered by
	// the user's selected ordering, so the frontend can display numbered bubbles.
	var qm []models.QuestionMapItem
	if mode == "REGULAR" {
		qm = buildQuestionMap(orderedScope, seenIDs)
	}

	return &StartResult{
		Session:             session,
		FirstQuestion:       firstQ,
		ThetaStart:          thetaStart,
		QuestionMap:         qm,
		TotalScopeQuestions: len(candidates),
	}, nil
}

func (s *LearnService) GetSession(ctx context.Context, userID, sessionID string) (*models.GetSessionResponse, error) {
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("learn: session not found: %w", err)
	}
	if session.UserID != userID {
		return nil, fmt.Errorf("learn: session belongs to different user")
	}
	if session.Status != "ACTIVE" {
		return nil, fmt.Errorf("learn: session is %q, not ACTIVE", session.Status)
	}

	responses, _ := session.Responses()

	resp := &models.GetSessionResponse{
		SessionID:        session.ID,
		Mode:             session.Mode,
		Chapter:          session.Chapter,
		ThetaStart:       float64(session.ThetaStart),
		ThetaCurrent:     float64(session.ThetaCurrent),
		QuestionCount:    len(responses),
		QuestionOrdering: session.QuestionOrdering,
		QuestionLimit:    session.QuestionLimit,
		BiometricEnabled: session.BiometricEnabled,
	}

	// Load scope candidates for question map (REGULAR) and total count (all modes).
	scope, err := session.Scope()
	if err == nil {
		chapters := scope.Chapters
		if len(chapters) == 0 {
			chapters = []string{session.Chapter}
		}
		candidates, err := s.loadQuestionsFiltered(ctx, chapters, scope.Years, scope.ExamTypes)
		if err == nil {
			resp.TotalScopeQuestions = len(candidates)
			if session.Mode == "REGULAR" {
				seenIDs, _ := s.loadSeenIDs(ctx, userID)
				if seenIDs == nil {
					seenIDs = make(map[string]int)
				}
				orderedScope := sortQuestionsByOrdering(candidates, session.QuestionOrdering)
				if len(orderedScope) > session.QuestionLimit {
					orderedScope = orderedScope[:session.QuestionLimit]
				}
				resp.QuestionMap = buildQuestionMap(orderedScope, seenIDs)
			}
		}
	}

	if session.CurrentQuestionID != nil {
		q, err := s.loadCurrentQuestion(ctx, session)
		if err == nil {
			qp := models.ToQuestionPayload(q)
			s.EnrichAttemptCount(ctx, session.UserID, &qp)
			resp.CurrentQuestion = &qp
		}
	}

	for _, r := range responses {
		q, err := s.loadQuestionByID(ctx, r.QuestionID)
		if err != nil {
			continue
		}
		qp := models.ToQuestionPayload(q)
		s.EnrichAttemptCount(ctx, session.UserID, &qp)
		resp.Responses = append(resp.Responses, models.ResponseHistoryItem{
			Question:        qp,
			IsCorrect:       r.IsCorrect,
			Skipped:         r.Skipped,
			SelectedOptions: r.SelectedOptions,
			CorrectOptions:  q.CorrectOptions(),
		})
	}

	return resp, nil
}

// SeekToQuestion navigates the session to a specific question in the ordered scope.
// Used when the user clicks a question in the full-scope question map.
// Does NOT modify responses — the seeked-over questions remain available.
func (s *LearnService) SeekToQuestion(ctx context.Context, userID, sessionID, questionID string) (*models.Question, error) {
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("learn: session not found: %w", err)
	}
	if session.UserID != userID {
		return nil, fmt.Errorf("learn: session belongs to different user")
	}
	if session.Status != "ACTIVE" {
		return nil, fmt.Errorf("learn: session is %q, not ACTIVE", session.Status)
	}

	question, err := s.loadQuestionByID(ctx, questionID)
	if err != nil {
		return nil, fmt.Errorf("learn: question not found: %w", err)
	}

	if err := s.sessionRepo.UpdateCurrentQuestion(ctx, sessionID, questionID); err != nil {
		return nil, fmt.Errorf("learn: update current question: %w", err)
	}

	return question, nil
}

type SubmitResult struct {
	IsCorrect      bool
	CorrectOptions []string
	ThetaBefore    float64
	ThetaAfter     float64
	NextQuestion   *models.Question
}

func (s *LearnService) Submit(ctx context.Context, userID, sessionID, questionID string,
	selectedOptions []string, timeTakenMs int64,
) (*SubmitResult, error) {
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("learn: session not found: %w", err)
	}
	if session.UserID != userID {
		return nil, fmt.Errorf("learn: session belongs to different user")
	}
	if session.Status != "ACTIVE" {
		return nil, fmt.Errorf("learn: session is %q, not ACTIVE", session.Status)
	}

	question, err := s.loadQuestionByID(ctx, questionID)
	if err != nil {
		return nil, fmt.Errorf("learn: question not found: %w", err)
	}

	correctOptions := question.CorrectOptions()
	isCorrect := optionsMatch(selectedOptions, correctOptions)

	thetaBefore := float64(session.ThetaCurrent)
	thetaAfter := ComputeThetaUpdate(thetaBefore, float64(question.GlickoRating), isCorrect, timeTakenMs, int64(question.TimespentAvgMs))

	responses, err := session.Responses()
	if err != nil {
		return nil, fmt.Errorf("learn: parse responses: %w", err)
	}

	consecutiveCorrect, consecutiveWrong := computeStreaks(responses, isCorrect)

	seenIDs, err := s.loadSeenIDs(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("learn: load seen: %w", err)
	}

	dna, err := s.loadDNA(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("learn: load dna: %w", err)
	}

	seenIDs[questionID] = seenIDs[questionID] + 1

	chapterAvgMs, _ := s.statsRepo.GetChapterAvgTimeMs(ctx, userID, question.Chapter)

	scState := &ScState{
		ThetaCurrent:          thetaAfter,
		Subject:               question.Subject,
		SubjectBias:           dna.SubjectBias[question.Subject],
		PeakPerformanceHour:   dna.PeakPerformanceHour,
		CurrentHour:           time.Now().Hour(),
		ConsecutiveCorrect:    consecutiveCorrect,
		ConsecutiveWrong:      consecutiveWrong,
		QuestionCount:         len(responses) + 1,
		AvgSessionLength:      dna.AvgSessionLength,
		AvgTimeTakenMs:        float64(dna.AvgTimeTakenMs),
		ChapterAvgTimeTakenMs: float64(chapterAvgMs),
		CarelessnessIndex:     dna.CarelessnessIndex,
		AttemptCounts:         seenIDs,
	}

	var nextQ *models.Question
	var pickResult *PickResult

	if session.Mode == "REGULAR" {
		sessionSeen := make(map[string]bool)
		for _, r := range responses {
			sessionSeen[r.QuestionID] = true
		}
		sessionSeen[questionID] = true
		nextQ = s.pickRegularQuestion(ctx, session, seenIDs, sessionSeen)
	} else {
		scope, _ := session.Scope()
		chapters := scope.Chapters
		if len(chapters) == 0 {
			chapters = []string{session.Chapter}
		}
		sessionSeen := make(map[string]bool)
		for _, r := range responses {
			sessionSeen[r.QuestionID] = true
		}
		sessionSeen[questionID] = true

		var available []models.Question

		// After a wrong answer, use pgvector HNSW index for fast embedding
		// similarity search instead of loading all candidates into Go memory.
		if !isCorrect && len(question.Embedding.Slice()) > 0 {
			similar, err2 := s.findSimilarCandidates(ctx, question.Embedding, chapters, scope.Years, scope.ExamTypes, questionID, 30)
			if err2 == nil {
				for _, q := range similar {
					if !sessionSeen[q.ID] {
						available = append(available, q)
					}
				}
			}
		}

		// Fall back to full candidate load (cold start / after correct / no embeddings)
		if len(available) == 0 {
			candidates, err2 := s.loadQuestionsFiltered(ctx, chapters, scope.Years, scope.ExamTypes)
			if err2 == nil && len(candidates) > 0 {
				for _, q := range candidates {
					if !sessionSeen[q.ID] {
						available = append(available, q)
					}
				}
			}
		}

		if len(available) > 0 {
			pickResult = PickBestQuestion(available, scState, question, isCorrect)
			nextQ = pickResult.Question
		}
	}

	var nextQID *string
	if nextQ != nil {
		nextQID = &nextQ.ID
	}

	resp := models.SessionResponse{
		QuestionID:      questionID,
		SelectedOptions: selectedOptions,
		IsCorrect:       isCorrect,
		ThetaBefore:     thetaBefore,
		ThetaAfter:      thetaAfter,
		QuestionCount:   len(responses) + 1,
		TimeTakenMs:     timeTakenMs,
		SubmittedAt:     time.Now(),
	}

	if pickResult != nil {
		resp.ScScore = pickResult.Scores.Total
		resp.DifficultyFit = pickResult.Scores.DifficultyFit
		resp.VectorSimilarity = pickResult.Scores.VectorSimilarity
		resp.TimeMatch = pickResult.Scores.TimeMatch
		resp.NoveltyFactor = pickResult.Scores.NoveltyFactor
		resp.ImmediateReinforce = pickResult.Scores.ImmediateReinforce
		resp.CarelessnessPenalty = pickResult.Scores.CarelessnessPenalty
		resp.ThetaEffective = pickResult.ThetaEff
		resp.Momentum = pickResult.Momentum
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("learn: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.sessionRepo.LockSessionForUpdate(ctx, tx, sessionID); err != nil {
		return nil, fmt.Errorf("learn: lock session: %w", err)
	}

	if err := s.questionStats.UpsertTx(ctx, tx, userID, questionID, isCorrect, timeTakenMs); err != nil {
		return nil, fmt.Errorf("learn: upsert stats: %w", err)
	}
	if err := s.sessionRepo.AppendResponseAndUpdateMetadataTx(ctx, tx, sessionID, resp,
		float32(thetaAfter), len(responses)+1, nextQID,
	); err != nil {
		return nil, fmt.Errorf("learn: append response: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("learn: commit tx: %w", err)
	}

	if err := s.questionRepo.UpdateQuestionStats(ctx, questionID, timeTakenMs, isCorrect); err != nil {
		slog.Warn("update question stats failed", "question_id", questionID, "error", err)
	}

	_ = s.cache.Del(ctx, fmt.Sprintf("seen:%s", userID))

	return &SubmitResult{
		IsCorrect:      isCorrect,
		CorrectOptions: correctOptions,
		ThetaBefore:    thetaBefore,
		ThetaAfter:     thetaAfter,
		NextQuestion:   nextQ,
	}, nil
}

type SkipResult struct {
	ThetaBefore  float64
	ThetaAfter   float64
	NextQuestion *models.Question
}

type FiltersResult struct {
	Years     []string `json:"years"`
	ExamTypes []string `json:"exam_types"`
}

func (s *LearnService) GetFilters(ctx context.Context) (*FiltersResult, error) {
	cacheKey := "filters"
	var cached FiltersResult
	if err := s.cache.Get(ctx, cacheKey, &cached); err == nil {
		return &cached, nil
	}

	years, err := s.questionRepo.GetAvailableYears(ctx)
	if err != nil {
		return nil, fmt.Errorf("learn: get years: %w", err)
	}
	examTypes, err := s.questionRepo.GetAvailableExamTypes(ctx)
	if err != nil {
		return nil, fmt.Errorf("learn: get exam types: %w", err)
	}
	result := &FiltersResult{Years: years, ExamTypes: examTypes}
	_ = s.cache.Set(ctx, cacheKey, result, 1*time.Hour)
	return result, nil
}

func (s *LearnService) Skip(ctx context.Context, userID, sessionID, questionID string, timeTakenMs int64) (*SkipResult, error) {
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("learn: session not found: %w", err)
	}
	if session.UserID != userID {
		return nil, fmt.Errorf("learn: session belongs to different user")
	}
	if session.Status != "ACTIVE" {
		return nil, fmt.Errorf("learn: session is %q, not ACTIVE", session.Status)
	}

	question, err := s.loadQuestionByID(ctx, questionID)
	if err != nil {
		return nil, fmt.Errorf("learn: question not found: %w", err)
	}

	thetaBefore := float64(session.ThetaCurrent)
	thetaAfter := thetaBefore

	responses, err := session.Responses()
	if err != nil {
		return nil, fmt.Errorf("learn: parse responses: %w", err)
	}

	consecutiveCorrect, consecutiveWrong := 0, 0

	seenIDs, err := s.loadSeenIDs(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("learn: load seen: %w", err)
	}

	dna, err := s.loadDNA(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("learn: load dna: %w", err)
	}

	seenIDs[questionID] = seenIDs[questionID] + 1

	chapterAvgMs, _ := s.statsRepo.GetChapterAvgTimeMs(ctx, userID, question.Chapter)

	scState := &ScState{
		ThetaCurrent:          thetaAfter,
		Subject:               question.Subject,
		SubjectBias:           dna.SubjectBias[question.Subject],
		PeakPerformanceHour:   dna.PeakPerformanceHour,
		CurrentHour:           time.Now().Hour(),
		ConsecutiveCorrect:    consecutiveCorrect,
		ConsecutiveWrong:      consecutiveWrong,
		QuestionCount:         len(responses) + 1,
		AvgSessionLength:      dna.AvgSessionLength,
		AvgTimeTakenMs:        float64(dna.AvgTimeTakenMs),
		ChapterAvgTimeTakenMs: float64(chapterAvgMs),
		CarelessnessIndex:     dna.CarelessnessIndex,
		AttemptCounts:         seenIDs,
	}

	var nextQ *models.Question
	var pickResult *PickResult

	if session.Mode == "REGULAR" {
		sessionSeen := make(map[string]bool)
		for _, r := range responses {
			sessionSeen[r.QuestionID] = true
		}
		sessionSeen[questionID] = true
		nextQ = s.pickRegularQuestion(ctx, session, seenIDs, sessionSeen)
	} else {
		scope, _ := session.Scope()
		chapters := scope.Chapters
		if len(chapters) == 0 {
			chapters = []string{session.Chapter}
		}
		sessionSeen := make(map[string]bool)
		for _, r := range responses {
			sessionSeen[r.QuestionID] = true
		}
		sessionSeen[questionID] = true

		var available []models.Question

		// After a skip (treated as wrong), use pgvector HNSW index for fast
		// embedding similarity search instead of loading all candidates.
		if len(question.Embedding.Slice()) > 0 {
			similar, err2 := s.findSimilarCandidates(ctx, question.Embedding, chapters, scope.Years, scope.ExamTypes, questionID, 30)
			if err2 == nil {
				for _, q := range similar {
					if !sessionSeen[q.ID] {
						available = append(available, q)
					}
				}
			}
		}

		// Fall back to full candidate load
		if len(available) == 0 {
			candidates, err2 := s.loadQuestionsFiltered(ctx, chapters, scope.Years, scope.ExamTypes)
			if err2 == nil && len(candidates) > 0 {
				for _, q := range candidates {
					if !sessionSeen[q.ID] {
						available = append(available, q)
					}
				}
			}
		}

		if len(available) > 0 {
			pickResult = PickAfterSkip(available, scState, question)
			nextQ = pickResult.Question
		}
	}

	var nextQID *string
	if nextQ != nil {
		nextQID = &nextQ.ID
	}

	resp := models.SessionResponse{
		QuestionID:      questionID,
		SelectedOptions: nil,
		IsCorrect:       false,
		Skipped:         true,
		ThetaBefore:     thetaBefore,
		ThetaAfter:      thetaAfter,
		QuestionCount:   len(responses) + 1,
		TimeTakenMs:     timeTakenMs,
		SubmittedAt:     time.Now(),
	}

	if pickResult != nil {
		resp.ScScore = pickResult.Scores.Total
		resp.DifficultyFit = pickResult.Scores.DifficultyFit
		resp.VectorSimilarity = pickResult.Scores.VectorSimilarity
		resp.TimeMatch = pickResult.Scores.TimeMatch
		resp.NoveltyFactor = pickResult.Scores.NoveltyFactor
		resp.ImmediateReinforce = pickResult.Scores.ImmediateReinforce
		resp.CarelessnessPenalty = pickResult.Scores.CarelessnessPenalty
		resp.ThetaEffective = pickResult.ThetaEff
		resp.Momentum = pickResult.Momentum
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("learn: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.sessionRepo.LockSessionForUpdate(ctx, tx, sessionID); err != nil {
		return nil, fmt.Errorf("learn: lock session: %w", err)
	}

	if err := s.sessionRepo.AppendResponseAndUpdateMetadataTx(ctx, tx, sessionID, resp,
		float32(thetaAfter), len(responses)+1, nextQID,
	); err != nil {
		return nil, fmt.Errorf("learn: append skip response: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("learn: commit tx: %w", err)
	}

	if err := s.questionRepo.UpdateQuestionStats(ctx, questionID, timeTakenMs, false); err != nil {
		slog.Warn("skip: update question stats failed", "question_id", questionID, "error", err)
	}

	_ = s.cache.Del(ctx, fmt.Sprintf("seen:%s", userID))

	return &SkipResult{
		ThetaBefore:  thetaBefore,
		ThetaAfter:   thetaAfter,
		NextQuestion: nextQ,
	}, nil
}

type CloseResult struct {
	SessionID      string
	Chapter        string
	ThetaStart     float64
	ThetaFinal     float64
	MasteryScore   float64
	TotalQuestions int
	CorrectCount   int
	Accuracy       float64
	AvgTimeTakenMs int64
}

func (s *LearnService) Close(ctx context.Context, userID, sessionID string) (*CloseResult, error) {
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("learn: session not found: %w", err)
	}
	if session.UserID != userID {
		return nil, fmt.Errorf("learn: session belongs to different user")
	}
	if session.Status != "ACTIVE" {
		return nil, fmt.Errorf("learn: session already %q", session.Status)
	}
	return s.closeSessionInternal(ctx, session)
}

// closeSessionInternal performs the core session close logic — computes stats,
// updates Glicko rating, chapter stats, DNA, and marks the session COMPLETED.
// It assumes the caller has verified the session is ACTIVE and belongs to the
// correct user. Used by both the public Close() API and the background stale
// session cleanup (ProcessStaleSessions).
func (s *LearnService) closeSessionInternal(ctx context.Context, session *models.LearnSession) (*CloseResult, error) {
	sessionID := session.ID
	userID := session.UserID

	responses, err := session.Responses()
	if err != nil {
		return nil, fmt.Errorf("learn: parse responses: %w", err)
	}

	totalQ := len(responses)
	if totalQ == 0 {
		_ = s.sessionRepo.Abandon(ctx, sessionID)
		return &CloseResult{
			SessionID: sessionID,
			Chapter:   session.Chapter,
		}, nil
	}

	correctCount := 0
	skipCount := 0
	var totalTimeMs int64
	for _, r := range responses {
		if r.Skipped {
			skipCount++
		} else if r.IsCorrect {
			correctCount++
		}
		totalTimeMs += r.TimeTakenMs
	}
	attemptedQ := totalQ - skipCount
	accuracy := 0.0
	if attemptedQ > 0 {
		accuracy = float64(correctCount) / float64(attemptedQ)
	}
	avgTimeMs := totalTimeMs / int64(totalQ)

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("learn: get user: %w", err)
	}
	dna, err := user.DNA()
	if err != nil {
		return nil, fmt.Errorf("learn: parse dna: %w", err)
	}

	avgOpponentRating := s.computeAvgOpponentRating(ctx, responses)
	playerRD := math.Max(30.0, float64(user.LearnRD))
	glickoOut := UpdateGlickoFromSession(GlickoSessionInput{
		PlayerRating:      float64(user.LearnRating),
		PlayerRD:          playerRD,
		PlayerVol:         float64(user.LearnVol),
		AvgOpponentRating: avgOpponentRating,
		Score:             accuracy,
	})

	thetaFinal := float64(session.ThetaCurrent)
	masteryScore := ComputeMasteryScore(thetaFinal)

	type chCounts struct {
		correct, total int
		totalTimeMs    int64
	}
	chapterCounts := make(map[string]chCounts)

	questionIDs := make([]string, len(responses))
	for i, r := range responses {
		questionIDs[i] = r.QuestionID
	}
	chaptersByID, err := s.loadQuestionChaptersBatch(ctx, questionIDs)
	if err != nil {
		return nil, fmt.Errorf("learn: load question chapters: %w", err)
	}

	for _, r := range responses {
		ch := chaptersByID[r.QuestionID]
		if ch == "" {
			continue
		}
		entry := chapterCounts[ch]
		entry.total++
		if r.IsCorrect {
			entry.correct++
		}
		entry.totalTimeMs += r.TimeTakenMs
		chapterCounts[ch] = entry
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("learn: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.sessionRepo.LockSessionForUpdate(ctx, tx, sessionID); err != nil {
		return nil, fmt.Errorf("learn: lock session: %w", err)
	}

	if err := s.userRepo.UpdateLearnRatingTx(ctx, tx, userID,
		float32(glickoOut.NewRating), float32(glickoOut.NewRD), float32(glickoOut.NewVol),
	); err != nil {
		return nil, fmt.Errorf("learn: update rating: %w", err)
	}

	for chapter, counts := range chapterCounts {
		chAvgTime := counts.totalTimeMs / int64(counts.total)
		isSingleChapter := len(chapterCounts) == 1
		existingCS, err2 := s.statsRepo.GetChapterStats(ctx, userID, chapter)
		cs := models.ChapterStats{
			GlickoRating:    glickoOut.NewRating,
			GlickoRD:        glickoOut.NewRD,
			GlickoVol:       glickoOut.NewVol,
			TotalAttempts:   counts.total,
			CorrectAttempts: counts.correct,
			AvgTimeMs:       chAvgTime,
			SessionsCount:   1,
			LastPracticedAt: time.Now(),
		}
		if isSingleChapter || (err2 != nil || existingCS == nil) {
			cs.Theta = thetaFinal
			cs.MasteryScore = masteryScore
		} else {
			cs.Theta = existingCS.Theta
			cs.MasteryScore = existingCS.MasteryScore
		}
		if err2 == nil && existingCS != nil {
			cs.TotalAttempts += existingCS.TotalAttempts
			cs.CorrectAttempts += existingCS.CorrectAttempts
			cs.SessionsCount += existingCS.SessionsCount
		}
		if err := s.statsRepo.UpsertChapterStatsTx(ctx, tx, userID, chapter, cs); err != nil {
			return nil, fmt.Errorf("learn: upsert chapter stats: %w", err)
		}
	}

	dna, err = s.recomputeDNA(ctx, userID, responses)
	if err != nil {
		return nil, fmt.Errorf("learn: recompute dna: %w", err)
	}
	if err := s.userRepo.UpdateLearningDNATx(ctx, tx, userID, dna); err != nil {
		return nil, fmt.Errorf("learn: update dna: %w", err)
	}

	if err := s.sessionRepo.CloseTx(ctx, tx, sessionID, float32(thetaFinal)); err != nil {
		return nil, fmt.Errorf("learn: close session: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("learn: commit tx: %w", err)
	}

	return &CloseResult{
		SessionID:      sessionID,
		Chapter:        session.Chapter,
		ThetaStart:     float64(session.ThetaStart),
		ThetaFinal:     thetaFinal,
		MasteryScore:   masteryScore,
		TotalQuestions: totalQ,
		CorrectCount:   correctCount,
		Accuracy:       accuracy,
		AvgTimeTakenMs: avgTimeMs,
	}, nil
}

// ProcessStaleSessions finds ACTIVE sessions that haven't been updated since
// the cutoff. Sessions with responses are properly closed (triggering Glicko
// rating update, chapter stats, DNA recompute). Sessions with no responses
// are abandoned. This runs in a background goroutine.
func (s *LearnService) ProcessStaleSessions(ctx context.Context, staleAge time.Duration) (int, error) {
	cutoff := time.Now().Add(-staleAge)
	sessions, err := s.sessionRepo.GetStaleActiveSessions(ctx, cutoff)
	if err != nil {
		return 0, fmt.Errorf("learn: fetch stale sessions: %w", err)
	}

	var closed int
	for _, sess := range sessions {
		// Use a per-session timeout so one slow close doesn't block others
		sessCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		result, err := s.closeSessionInternal(sessCtx, sess)
		cancel()
		if err != nil {
			slog.Warn("failed to close stale session",
				"session_id", sess.ID,
				"user_id", sess.UserID,
				"error", err,
			)
			continue
		}
		closed++
		if result.TotalQuestions > 0 {
			slog.Info("closed stale session",
				"session_id", sess.ID,
				"user_id", sess.UserID,
				"questions", result.TotalQuestions,
			)
		} else {
			slog.Info("abandoned empty stale session",
				"session_id", sess.ID,
				"user_id", sess.UserID,
			)
		}
	}
	return closed, nil
}

func (s *LearnService) computeAvgOpponentRating(ctx context.Context, responses []models.SessionResponse) float64 {
	if len(responses) == 0 {
		return 1500.0
	}
	questionIDs := make([]string, len(responses))
	for i, r := range responses {
		questionIDs[i] = r.QuestionID
	}

	rows, err := s.pool.Query(ctx, `SELECT glicko_rating FROM questions WHERE id = ANY($1)`, questionIDs)
	if err != nil {
		return 1500.0
	}
	defer rows.Close()

	var total float64
	var count int
	for rows.Next() {
		var rating float32
		if err := rows.Scan(&rating); err != nil {
			continue
		}
		total += float64(rating)
		count++
	}

	if count == 0 {
		return 1500.0
	}
	return total / float64(count)
}

func (s *LearnService) buildScState(ctx context.Context, userID string, theta float64, subject string) *ScState {
	dna, err := s.loadDNA(ctx, userID)
	if err != nil {
		return &ScState{ThetaCurrent: theta, AvgTimeTakenMs: 60000}
	}

	var subjectBias float64
	if subject != "" && dna.SubjectBias != nil {
		subjectBias = dna.SubjectBias[subject]
	}

	return &ScState{
		ThetaCurrent:        theta,
		Subject:             subject,
		SubjectBias:         subjectBias,
		PeakPerformanceHour: dna.PeakPerformanceHour,
		CurrentHour:         time.Now().Hour(),
		AvgSessionLength:    dna.AvgSessionLength,
		AvgTimeTakenMs:      float64(dna.AvgTimeTakenMs),
		CarelessnessIndex:   dna.CarelessnessIndex,
	}
}

func (s *LearnService) PickSimilarQuestion(ctx context.Context, userID, sessionID, questionID string) (*models.Question, error) {
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("learn: session not found: %w", err)
	}
	if session.UserID != userID {
		return nil, fmt.Errorf("learn: session belongs to different user")
	}
	if session.Status != "ACTIVE" {
		return nil, fmt.Errorf("learn: session is %q, not ACTIVE", session.Status)
	}

	currentQ, err := s.loadQuestionByID(ctx, questionID)
	if err != nil {
		return nil, fmt.Errorf("learn: question not found: %w", err)
	}

	scope, err := session.Scope()
	if err != nil {
		return nil, fmt.Errorf("learn: parse scope: %w", err)
	}
	chapters := scope.Chapters
	if len(chapters) == 0 {
		chapters = []string{session.Chapter}
	}

	candidates, err := s.loadQuestionsFiltered(ctx, chapters, scope.Years, scope.ExamTypes)
	if err != nil {
		return nil, fmt.Errorf("learn: load candidates: %w", err)
	}

	responses, err := session.Responses()
	if err != nil {
		return nil, fmt.Errorf("learn: load responses: %w", err)
	}

	sessionSeen := make(map[string]bool)
	for _, r := range responses {
		sessionSeen[r.QuestionID] = true
	}
	sessionSeen[questionID] = true

	var available []models.Question
	for _, q := range candidates {
		if !sessionSeen[q.ID] {
			available = append(available, q)
		}
	}

	if len(available) == 0 {
		return nil, fmt.Errorf("learn: no more available questions in session scope")
	}

	currentEmb := currentQ.Embedding.Slice()
	var best *models.Question
	var bestScore float64 = -1

	// Use pgvector HNSW index when the current question has an embedding.
	if len(currentEmb) > 0 {
		similar, err2 := s.questionRepo.FindSimilarByChapters(ctx, currentQ.Embedding, chapters, questionID, 10)
		if err2 == nil && len(similar) > 0 {
			similar = filterByScope(similar, scope.Years, scope.ExamTypes)
			// Pick the first available (not session-seen) from the pgvector results
			seenSet := make(map[string]bool, len(available))
			for _, a := range available {
				seenSet[a.ID] = true
			}
			for _, q := range similar {
				if seenSet[q.ID] {
					best = &q
					break
				}
			}
		}
	}

	// Fallback: brute-force similarity on the available set
	if best == nil {
		for i := range available {
			q := &available[i]
			sim := 0.0
			if len(currentEmb) > 0 {
				qEmb := q.Embedding.Slice()
				sim = cosineSimilarity(qEmb, currentEmb)
				sim = (sim + 1) / 2
			}
			if sim > bestScore {
				bestScore = sim
				best = q
			}
		}
	}

	if best == nil {
		return nil, fmt.Errorf("learn: failed to find a similar question")
	}

	err = s.sessionRepo.UpdateCurrentQuestion(ctx, session.ID, best.ID)
	if err != nil {
		return nil, fmt.Errorf("learn: failed to update session current question: %w", err)
	}

	return best, nil
}

func ComputeMasteryScore(theta float64) float64 {
	// Map from theta range (800-3500, matching theta.go) to 0-100%.
	const floor, ceiling = 800.0, 3500.0
	if theta <= floor {
		return 0.0
	}
	if theta >= ceiling {
		return 100.0
	}
	return math.Round((theta-floor)/(ceiling-floor)*100*10) / 10
}

func computeStreaks(
	responses []models.SessionResponse,
	latestCorrect bool,
) (consecutiveCorrect, consecutiveWrong int) {
	streak := 1
	for i := len(responses) - 1; i >= 0; i-- {
		if responses[i].Skipped {
			continue
		}
		if responses[i].IsCorrect == latestCorrect {
			streak++
		} else {
			break
		}
	}
	if latestCorrect {
		return streak, 0
	}
	return 0, streak
}

func optionsMatch(selected, correct []string) bool {
	if len(selected) != len(correct) {
		return false
	}
	selectedSet := make(map[string]struct{}, len(selected))
	for _, s := range selected {
		selectedSet[strings.TrimSpace(s)] = struct{}{}
	}
	for _, c := range correct {
		c = strings.TrimSpace(c)
		if strings.Contains(c, "TO") {
			matched := false
			for sel := range selectedSet {
				if numericalMatch(sel, c) {
					matched = true
					break
				}
			}
			if !matched {
				return false
			}
		} else {
			if _, ok := selectedSet[c]; !ok {
				return false
			}
		}
	}
	return true
}

func numericalMatch(selected, rangeStr string) bool {
	selVal, err := strconv.ParseFloat(strings.TrimSpace(selected), 64)
	if err != nil {
		return false
	}

	ranges := strings.Split(rangeStr, "OR")
	for _, r := range ranges {
		parts := strings.Split(r, "TO")
		if len(parts) != 2 {
			continue
		}
		lo, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		hi, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err1 != nil || err2 != nil {
			continue
		}
		if selVal >= lo && selVal <= hi {
			return true
		}
	}
	return false
}

// findSimilarCandidates returns the top-N questions across chapters ordered by
// embedding similarity to the given vector, using the pgvector HNSW index.
// Falls back to loading all questions (by chapter) if the pgvector query fails
// or returns too few results.
func (s *LearnService) findSimilarCandidates(ctx context.Context, emb pgvector.Vector, chapters, years, examTypes []string, excludeID string, limit int) ([]models.Question, error) {
	similar, err := s.questionRepo.FindSimilarByChapters(ctx, emb, chapters, excludeID, limit)
	if err == nil && len(similar) >= 3 {
		similar = filterByScope(similar, years, examTypes)
		if len(similar) >= 3 {
			applyDynamicRDToQuestions(similar)
			return similar, nil
		}
	}

	// Fallback: load all and sort by similarity in Go
	all, err2 := s.loadQuestionsFiltered(ctx, chapters, years, examTypes)
	if err2 != nil {
		if err != nil {
			return nil, fmt.Errorf("learn: find similar (%w) and fallback (%w)", err, err2)
		}
		return nil, err2
	}
	if len(similar) > 0 {
		// Prefer pgvector results, then supplement
		seen := make(map[string]bool, len(similar))
		for _, q := range similar {
			seen[q.ID] = true
		}
		currentEmb := emb.Slice()
		for _, q := range all {
			if !seen[q.ID] {
				qEmb := q.Embedding.Slice()
				if len(qEmb) > 0 && len(currentEmb) > 0 {
					sim := cosineSimilarity(qEmb, currentEmb)
					if sim > 0.5 {
						similar = append(similar, q)
					}
				}
			}
		}
		if len(similar) > limit {
			similar = similar[:limit]
		}
		return similar, nil
	}
	return all, nil
}

// sortQuestionsByOrdering sorts a slice of questions based on the given ordering.
// "latest_first"  — newest shift_date first (descending).
// "oldest_first"  — oldest shift_date first (ascending).
// "random"        — shuffled (Fisher-Yates). Default for unknown values.
func sortQuestionsByOrdering(questions []models.Question, ordering string) []models.Question {
	if len(questions) <= 1 {
		return questions
	}
	result := make([]models.Question, len(questions))
	copy(result, questions)

	switch ordering {
	case "latest_first":
		sort.SliceStable(result, func(i, j int) bool {
			return result[i].ShiftDate > result[j].ShiftDate
		})
	case "oldest_first":
		sort.SliceStable(result, func(i, j int) bool {
			return result[i].ShiftDate < result[j].ShiftDate
		})
	default: // "random" or unknown → shuffle
		rand.Shuffle(len(result), func(i, j int) {
			result[i], result[j] = result[j], result[i]
		})
	}
	return result
}

// buildQuestionMap extracts lightweight question map items (id + year + seen)
// from a question slice. Used in REGULAR mode to render the full-scope question map.
func buildQuestionMap(questions []models.Question, seenIDs map[string]int) []models.QuestionMapItem {
	items := make([]models.QuestionMapItem, 0, len(questions))
	for _, q := range questions {
		year := ""
		if len(q.ShiftDate) >= 4 {
			year = q.ShiftDate[:4]
		}
		_, seen := seenIDs[q.ID]
		items = append(items, models.QuestionMapItem{
			ID:   q.ID,
			Year: year,
			Seen: seen,
		})
	}
	return items
}

// pickRegularQuestion picks the next question for a REGULAR mode session.
// It uses the session's full scope (chapters, years, exam types) and ordering.
// Returns nil when all questions in scope have been attempted.
func (s *LearnService) pickRegularQuestion(ctx context.Context, session *models.LearnSession, seenIDs map[string]int, sessionSeen map[string]bool) *models.Question {
	scope, err := session.Scope()
	if err != nil {
		return nil
	}
	chapters := scope.Chapters
	if len(chapters) == 0 {
		chapters = []string{session.Chapter}
	}

	// Load all questions in scope chapters.
	candidates, err := s.loadQuestionsFiltered(ctx, chapters, scope.Years, scope.ExamTypes)
	if err != nil || len(candidates) == 0 {
		return nil
	}

	// Sort by the user's chosen ordering and trim to the session's question limit.
	ordered := sortQuestionsByOrdering(candidates, session.QuestionOrdering)
	if len(ordered) > session.QuestionLimit {
		ordered = ordered[:session.QuestionLimit]
	}

	// Find the first question within the limited pool that hasn't been seen
	// (all-time or within this session).
	for _, q := range ordered {
		if _, seen := seenIDs[q.ID]; seen {
			continue
		}
		if sessionSeen != nil && sessionSeen[q.ID] {
			continue
		}
		return &q
	}
	return nil
}

func filterUnseen(questions []models.Question, seenIDs map[string]int) []models.Question {
	var unseen []models.Question
	for _, q := range questions {
		if _, seen := seenIDs[q.ID]; !seen {
			unseen = append(unseen, q)
		}
	}
	return unseen
}

func filterByScope(questions []models.Question, years, examTypes []string) []models.Question {
	if len(years) == 0 && len(examTypes) == 0 {
		return questions
	}
	var filtered []models.Question
	for _, q := range questions {
		if len(examTypes) > 0 {
			match := false
			for _, et := range examTypes {
				if q.ExamType == et {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}
		if len(years) > 0 {
			match := false
			for _, y := range years {
				if strings.HasPrefix(q.ShiftDate, y) {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}
		filtered = append(filtered, q)
	}
	return filtered
}

var (
	idCounter  int64
	instanceID = func() string {
		h, err := os.Hostname()
		if err != nil {
			return "unknown"
		}
		return h
	}()
)

func generateID() string {
	b := make([]byte, 8)
	if _, err := cryptorand.Read(b); err != nil {
		n := atomic.AddInt64(&idCounter, 1)
		ts := time.Now().UnixMilli()
		return fmt.Sprintf("sess_%s_%d_%d", instanceID, ts, n)
	}
	return fmt.Sprintf("sess_%s", hex.EncodeToString(b))
}
