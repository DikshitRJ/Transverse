package services

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"transverse/internal/cache"
	"transverse/internal/models"
	"transverse/internal/repository"
)

// PracticeService coordinates adaptive session state transitions, problem recommendations,
// psychometric updates, and submission evaluations.
type PracticeService struct {
	problemRepo *repository.ProblemRepo
	statsRepo   *repository.StatsRepo
	sessionRepo *repository.SessionRepo
	userRepo    *repository.UserRepo
	probStats   *repository.ProblemStatsRepo
	graphSvc    *GraphService
	cache       cache.Cache
	pool        *pgxpool.Pool
	judge0      *Judge0Service
}

// NewPracticeService constructs a new PracticeService with all required dependencies.
func NewPracticeService(
	problemRepo *repository.ProblemRepo,
	statsRepo *repository.StatsRepo,
	sessionRepo *repository.SessionRepo,
	userRepo *repository.UserRepo,
	probStats *repository.ProblemStatsRepo,
	graphSvc *GraphService,
	cache cache.Cache,
	pool *pgxpool.Pool,
	judge0 *Judge0Service,
) *PracticeService {
	return &PracticeService{
		problemRepo: problemRepo,
		statsRepo:   statsRepo,
		sessionRepo: sessionRepo,
		userRepo:    userRepo,
		probStats:   probStats,
		graphSvc:    graphSvc,
		cache:       cache,
		pool:        pool,
		judge0:      judge0,
	}
}

// StartSession initiates or resumes an adaptive practice session for the specified user and scope.
func (ps *PracticeService) StartSession(ctx context.Context, userID string, req models.StartSessionRequest) (*models.StartSessionResponse, error) {
	// 1. Resolve canonical scope topics via knowledge graph
	resolvedTopics, err := ps.graphSvc.ResolveScope(req.Scope.Topics)
	if err != nil {
		return nil, fmt.Errorf("practice_service: resolve topics: %w", err)
	}
	req.Scope.Topics = resolvedTopics

	// 2. Check for an existing ACTIVE session for this user
	activeSess, err := ps.sessionRepo.GetActiveByUser(ctx, userID)
	if err == nil && activeSess != nil {
		var currPayload *models.ProblemPayload
		if activeSess.CurrentProblemID != nil {
			if currProb, err := ps.problemRepo.GetByID(ctx, *activeSess.CurrentProblemID); err == nil && currProb != nil {
				payload := models.ToProblemPayload(currProb)
				currPayload = &payload
			}
		}
		return &models.StartSessionResponse{
			SessionID:      activeSess.ID,
			Mode:           activeSess.Mode,
			Theta:          activeSess.ThetaCurrent,
			CurrentProblem: currPayload,
			Status:         activeSess.Status,
			CreatedAt:      activeSess.CreatedAt,
		}, nil
	}

	// 3. Ensure user exists
	user, err := ps.userRepo.GetOrCreate(ctx, userID, "", "")
	if err != nil {
		return nil, fmt.Errorf("practice_service: get or create user: %w", err)
	}

	dna, err := user.DNA()
	if err != nil {
		dna = models.DefaultDNA()
	}

	// 4. Determine baseline theta
	baselineTheta := user.Theta
	if len(req.Scope.Topics) > 0 {
		if tStat, _ := ps.statsRepo.GetByUserAndTopic(ctx, userID, req.Scope.Topics[0]); tStat != nil {
			baselineTheta = tStat.Theta
		}
	}
	if baselineTheta <= 0 {
		baselineTheta = 1500.0
	}

	// 5. Load candidate problems in scope
	candidates, err := ps.problemRepo.GetByScope(ctx, req.Scope)
	if err != nil || len(candidates) == 0 {
		// Fallback to topic search if scope is empty
		if len(req.Scope.Topics) > 0 {
			candidates, _ = ps.problemRepo.GetByTopic(ctx, req.Scope.Topics[0])
		}
		if len(candidates) == 0 {
			return nil, fmt.Errorf("practice_service: no problems found for requested scope")
		}
	}

	// 6. Load attempt counts
	attemptCounts, _ := ps.probStats.GetAttemptCountsByUser(ctx, userID)

	primaryTopic := ""
	if len(req.Scope.Topics) > 0 {
		primaryTopic = req.Scope.Topics[0]
	}

	state := ScState{
		ThetaCurrent:        baselineTheta,
		Topic:               primaryTopic,
		TopicBias:           dna.TopicBias[primaryTopic],
		ConsecutiveCorrect:  0,
		ConsecutiveWrong:    0,
		QuestionCount:       0,
		AvgSessionLength:    dna.AvgSessionLength,
		AvgTimeTakenMs:      float64(dna.AvgTimeTakenMs),
		CarelessnessIndex:   dna.CarelessnessIndex,
		AttemptCounts:       attemptCounts,
		ActiveSources:       req.Scope.Sources,
		SessionSourceCounts: make(map[string]int),
	}

	// 7. Cold start: pick best initial problem
	pick := PickBestProblem(candidates, state, nil, false)
	if pick.Problem == nil {
		return nil, fmt.Errorf("practice_service: failed to select initial problem")
	}

	sessionID := generateID("sess_")
	mode := req.Mode
	if mode == "" {
		mode = "ADAPTIVE"
	}

	sessionRecord := &models.PracticeSession{
		ID:               sessionID,
		UserID:           userID,
		Mode:             mode,
		ScopeRaw:         marshalJSON(req.Scope),
		ThetaStart:       baselineTheta,
		ThetaCurrent:     baselineTheta,
		CurrentProblemID: &pick.Problem.ID,
		ResponsesRaw:     []byte("[]"),
		QuestionCount:    0,
		Status:           "ACTIVE",
	}

	if err := ps.sessionRepo.Create(ctx, sessionRecord); err != nil {
		return nil, fmt.Errorf("practice_service: create session: %w", err)
	}

	payload := models.ToProblemPayload(pick.Problem)
	return &models.StartSessionResponse{
		SessionID:      sessionID,
		Mode:           mode,
		Theta:          baselineTheta,
		CurrentProblem: &payload,
		Status:         "ACTIVE",
		CreatedAt:      time.Now(),
	}, nil
}

// SubmitAnswer evaluates a code execution verdict from Judge0, updates the IRT ability estimate,
// records attempt metrics, and selects the next problem candidate.
func (ps *PracticeService) SubmitAnswer(ctx context.Context, userID, sessionID, judge0Token string, timeTakenMs int64) (*models.SubmitResponse, error) {
	session, err := ps.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("practice_service: get session: %w", err)
	}

	if session.UserID != userID {
		return nil, fmt.Errorf("practice_service: unauthorized session access")
	}

	if session.Status != "ACTIVE" {
		return nil, fmt.Errorf("practice_service: session is %s, cannot submit", session.Status)
	}

	if session.CurrentProblemID == nil {
		return nil, fmt.Errorf("practice_service: no active problem in session")
	}

	currentProblem, err := ps.problemRepo.GetByID(ctx, *session.CurrentProblemID)
	if err != nil {
		return nil, fmt.Errorf("practice_service: get current problem: %w", err)
	}

	// 1. Fetch verdict from Judge0
	verdict, err := ps.judge0.GetVerdict(ctx, judge0Token)
	if err != nil {
		return nil, fmt.Errorf("practice_service: get judge0 verdict: %w", err)
	}

	isCorrect := IsAccepted(verdict.StatusID)

	if timeTakenMs <= 0 {
		timeTakenMs = int64(verdict.TimeMs)
		if timeTakenMs <= 0 {
			timeTakenMs = 60000 // 1 min fallback
		}
	}

	// 2. IRT Theta update
	expectedTime := int64(currentProblem.AvgTimeMs)
	if expectedTime <= 0 {
		expectedTime = 120000 // 2 min default
	}

	newTheta := UpdateTheta(session.ThetaCurrent, currentProblem.GlickoRating, isCorrect, timeTakenMs, expectedTime)

	// 3. Record attempt counters
	_ = ps.probStats.RecordAttempt(ctx, userID, currentProblem.ID, isCorrect, timeTakenMs)
	_ = ps.problemRepo.IncrementAttemptCount(ctx, currentProblem.ID, isCorrect)

	// 4. Load past session responses to compute momentum and streaks
	pastResponses, _ := session.Responses()
	consecCorrect, consecWrong := ComputeSessionStreaks(pastResponses)
	if isCorrect {
		consecCorrect++
		consecWrong = 0
	} else {
		consecWrong++
		consecCorrect = 0
	}

	// 5. Build ScState for next problem selection
	user, _ := ps.userRepo.GetByID(ctx, userID)
	dna := models.DefaultDNA()
	if user != nil {
		dna, _ = user.DNA()
	}

	attemptCounts, _ := ps.probStats.GetAttemptCountsByUser(ctx, userID)
	scope, _ := session.Scope()

	sessionSourceCounts := make(map[string]int)
	for _, r := range pastResponses {
		if pastProb, err := ps.problemRepo.GetByID(ctx, r.ProblemID); err == nil && pastProb != nil {
			sessionSourceCounts[pastProb.Source]++
		}
	}
	sessionSourceCounts[currentProblem.Source]++

	activeTopic := currentProblem.Topic
	if len(scope.Topics) > 0 {
		activeTopic = scope.Topics[0]
	}

	state := ScState{
		ThetaCurrent:        newTheta,
		Topic:               activeTopic,
		TopicBias:           dna.TopicBias[activeTopic],
		ConsecutiveCorrect:  consecCorrect,
		ConsecutiveWrong:    consecWrong,
		QuestionCount:       session.QuestionCount + 1,
		AvgSessionLength:    dna.AvgSessionLength,
		AvgTimeTakenMs:      float64(dna.AvgTimeTakenMs),
		CarelessnessIndex:   dna.CarelessnessIndex,
		AttemptCounts:       attemptCounts,
		ActiveSources:       scope.Sources,
		SessionSourceCounts: sessionSourceCounts,
	}

	// 6. Select candidate pool: if wrong and embedding available, use ANN similar search
	var candidates []models.Problem
	if !isCorrect && len(currentProblem.Embedding.Slice()) > 0 {
		candidates, _ = ps.problemRepo.FindSimilar(ctx, currentProblem.Embedding, currentProblem.Topic, 30)
	}
	if len(candidates) == 0 {
		candidates, _ = ps.problemRepo.GetByScope(ctx, scope)
	}
	if len(candidates) == 0 {
		candidates, _ = ps.problemRepo.GetByTopic(ctx, activeTopic)
	}

	// Filter out problems already answered in this session
	answeredInSession := make(map[string]bool)
	for _, resp := range pastResponses {
		answeredInSession[resp.ProblemID] = true
	}
	answeredInSession[currentProblem.ID] = true

	filteredCandidates := make([]models.Problem, 0, len(candidates))
	for _, c := range candidates {
		if !answeredInSession[c.ID] {
			filteredCandidates = append(filteredCandidates, c)
		}
	}
	if len(filteredCandidates) == 0 {
		filteredCandidates = candidates
	}

	// 7. Pick best next problem
	pick := PickBestProblem(filteredCandidates, state, currentProblem, isCorrect)

	// 8. Record response
	respRecord := models.SessionResponse{
		ProblemID:           currentProblem.ID,
		IsCorrect:           isCorrect,
		Skipped:             false,
		Judge0StatusID:      verdict.StatusID,
		Judge0StatusDesc:    verdict.StatusDesc,
		ExecutionTimeMs:     verdict.TimeMs,
		MemoryKB:            verdict.MemoryKB,
		TimeTakenMs:         timeTakenMs,
		ThetaBefore:         session.ThetaCurrent,
		ThetaAfter:          newTheta,
		QuestionCount:       session.QuestionCount + 1,
		ScScore:             pick.Scores.Total,
		DifficultyFit:       pick.Scores.DifficultyFit,
		ConceptSimilarity:   pick.Scores.ConceptSimilarity,
		TopicProgression:    pick.Scores.TopicProgression,
		NoveltyFactor:       pick.Scores.NoveltyFactor,
		ImmediateReinforce:  pick.Scores.ImmediateReinforce,
		PlatformDiversity:   pick.Scores.PlatformDiversity,
		CarelessnessPenalty: pick.Scores.CarelessnessPenalty,
		ThetaEffective:      pick.ThetaEff,
		Momentum:            pick.Momentum,
		SubmittedAt:         time.Now(),
	}

	var nextProblemID *string
	var nextPayload *models.ProblemPayload
	if pick.Problem != nil {
		nextProblemID = &pick.Problem.ID
		p := models.ToProblemPayload(pick.Problem)
		nextPayload = &p
	}

	if err := ps.sessionRepo.AppendResponse(ctx, sessionID, respRecord, nextProblemID, newTheta); err != nil {
		return nil, fmt.Errorf("practice_service: append response: %w", err)
	}

	return &models.SubmitResponse{
		IsCorrect:           isCorrect,
		Verdict:             verdict,
		ThetaBefore:         session.ThetaCurrent,
		ThetaAfter:          newTheta,
		NextProblem:         nextPayload,
		SessionStatus:       "ACTIVE",
		QuestionCount:       session.QuestionCount + 1,
		CarelessnessPenalty: pick.Scores.CarelessnessPenalty,
	}, nil
}

// SkipProblem skips the current problem in the session, penalizes ability estimate accordingly,
// and selects the next problem.
func (ps *PracticeService) SkipProblem(ctx context.Context, userID, sessionID string, timeTakenMs int64) (*models.SkipResponse, error) {
	session, err := ps.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("practice_service: get session: %w", err)
	}

	if session.UserID != userID {
		return nil, fmt.Errorf("practice_service: unauthorized session access")
	}

	if session.Status != "ACTIVE" {
		return nil, fmt.Errorf("practice_service: session is %s, cannot skip", session.Status)
	}

	if session.CurrentProblemID == nil {
		return nil, fmt.Errorf("practice_service: no active problem to skip")
	}

	currentProblem, err := ps.problemRepo.GetByID(ctx, *session.CurrentProblemID)
	if err != nil {
		return nil, fmt.Errorf("practice_service: get current problem: %w", err)
	}

	if timeTakenMs <= 0 {
		timeTakenMs = 30000 // 30s default for skip
	}

	// IRT update: treat skip as incorrect with expected time
	expectedTime := int64(currentProblem.AvgTimeMs)
	if expectedTime <= 0 {
		expectedTime = 120000
	}
	newTheta := UpdateTheta(session.ThetaCurrent, currentProblem.GlickoRating, false, timeTakenMs, expectedTime)

	_ = ps.probStats.RecordAttempt(ctx, userID, currentProblem.ID, false, timeTakenMs)

	pastResponses, _ := session.Responses()
	consecCorrect, consecWrong := ComputeSessionStreaks(pastResponses)
	consecWrong++
	consecCorrect = 0

	user, _ := ps.userRepo.GetByID(ctx, userID)
	dna := models.DefaultDNA()
	if user != nil {
		dna, _ = user.DNA()
	}

	attemptCounts, _ := ps.probStats.GetAttemptCountsByUser(ctx, userID)
	scope, _ := session.Scope()

	activeTopic := currentProblem.Topic
	if len(scope.Topics) > 0 {
		activeTopic = scope.Topics[0]
	}

	state := ScState{
		ThetaCurrent:       newTheta,
		Topic:              activeTopic,
		TopicBias:          dna.TopicBias[activeTopic],
		ConsecutiveCorrect: consecCorrect,
		ConsecutiveWrong:   consecWrong,
		QuestionCount:      session.QuestionCount + 1,
		AvgSessionLength:   dna.AvgSessionLength,
		AvgTimeTakenMs:     float64(dna.AvgTimeTakenMs),
		CarelessnessIndex:  dna.CarelessnessIndex,
		AttemptCounts:      attemptCounts,
		ActiveSources:      scope.Sources,
	}

	candidates, _ := ps.problemRepo.GetByScope(ctx, scope)
	if len(candidates) == 0 {
		candidates, _ = ps.problemRepo.GetByTopic(ctx, activeTopic)
	}

	pick := PickBestProblem(candidates, state, currentProblem, false)

	respRecord := models.SessionResponse{
		ProblemID:      currentProblem.ID,
		IsCorrect:      false,
		Skipped:        true,
		TimeTakenMs:    timeTakenMs,
		ThetaBefore:    session.ThetaCurrent,
		ThetaAfter:     newTheta,
		QuestionCount:  session.QuestionCount + 1,
		ThetaEffective: pick.ThetaEff,
		Momentum:       pick.Momentum,
		SubmittedAt:    time.Now(),
	}

	var nextProblemID *string
	var nextPayload *models.ProblemPayload
	if pick.Problem != nil {
		nextProblemID = &pick.Problem.ID
		p := models.ToProblemPayload(pick.Problem)
		nextPayload = &p
	}

	if err := ps.sessionRepo.AppendResponse(ctx, sessionID, respRecord, nextProblemID, newTheta); err != nil {
		return nil, fmt.Errorf("practice_service: append skip response: %w", err)
	}

	return &models.SkipResponse{
		Skipped:       true,
		ThetaBefore:   session.ThetaCurrent,
		ThetaAfter:    newTheta,
		NextProblem:   nextPayload,
		QuestionCount: session.QuestionCount + 1,
	}, nil
}

// CloseSession finalizes an active practice session, executes Glicko-2 psychometric re-evaluation,
// recomputes LearningDNA, and persists mastery updates across topics.
func (ps *PracticeService) CloseSession(ctx context.Context, userID, sessionID string) (*models.CloseSessionResponse, error) {
	session, err := ps.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("practice_service: get session: %w", err)
	}

	if session.UserID != userID {
		return nil, fmt.Errorf("practice_service: unauthorized session access")
	}

	user, err := ps.userRepo.GetOrCreate(ctx, userID, "", "")
	if err != nil {
		return nil, fmt.Errorf("practice_service: get user: %w", err)
	}

	responses, _ := session.Responses()
	totalQuestions := len(responses)
	totalSolved := 0
	topicAttempts := make(map[string]int)
	topicCorrect := make(map[string]int)

	for _, r := range responses {
		if r.IsCorrect && !r.Skipped {
			totalSolved++
		}
		if p, err := ps.problemRepo.GetByID(ctx, r.ProblemID); err == nil && p != nil {
			topicAttempts[p.Topic]++
			if r.IsCorrect && !r.Skipped {
				topicCorrect[p.Topic]++
			}
		}
	}

	accuracy := 0.0
	if totalQuestions > 0 {
		accuracy = float64(totalSolved) / float64(totalQuestions)
	}

	masteryScore := CalculateMasteryScore(session.ThetaCurrent)

	// Glicko-2 update
	gameOutcomes := make([]GameOutcome, 0, totalQuestions)
	for _, r := range responses {
		if p, err := ps.problemRepo.GetByID(ctx, r.ProblemID); err == nil && p != nil {
			score := 0.0
			if r.IsCorrect && !r.Skipped {
				score = 1.0
			}
			gameOutcomes = append(gameOutcomes, GameOutcome{
				OpponentRating: p.GlickoRating,
				OpponentRD:     p.GlickoRD,
				Score:          score,
			})
		}
	}

	newRating, newRD, newVol := UpdateGlicko2(user.GlickoRating, user.GlickoRD, user.GlickoVol, gameOutcomes)

	// Update topic stats
	perTopicBreakdown := make(map[string]models.TopicProgress)
	for topic, attempts := range topicAttempts {
		correct := topicCorrect[topic]
		existingStat, _ := ps.statsRepo.GetByUserAndTopic(ctx, userID, topic)

		tTheta := session.ThetaCurrent
		tMastery := masteryScore
		tGlicko := newRating
		tAttempts := attempts
		tCorrect := correct

		if existingStat != nil {
			tAttempts += existingStat.AttemptCount
			tCorrect += existingStat.CorrectCount
		}

		statRecord := &models.TopicStats{
			UserID:       userID,
			Topic:        topic,
			Theta:        tTheta,
			MasteryScore: tMastery,
			GlickoRating: tGlicko,
			AttemptCount: tAttempts,
			CorrectCount: tCorrect,
		}
		_ = ps.statsRepo.Upsert(ctx, statRecord)

		perTopicBreakdown[topic] = models.TopicProgress{
			Topic:        topic,
			MasteryScore: tMastery,
			Theta:        tTheta,
			GlickoRating: tGlicko,
			AttemptCount: tAttempts,
			CorrectCount: tCorrect,
		}
	}

	// Update LearningDNA
	dna, _ := user.DNA()
	sessionDuration := time.Since(session.CreatedAt)
	updatedDNA := UpdateLearningDNA(dna, responses, sessionDuration)

	dnaBytes := marshalJSON(updatedDNA)
	user.Theta = session.ThetaCurrent
	user.GlickoRating = newRating
	user.GlickoRD = newRD
	user.GlickoVol = newVol
	user.DNARaw = dnaBytes
	_ = ps.userRepo.Update(ctx, user)

	// Mark session COMPLETED
	_ = ps.sessionRepo.CloseSession(ctx, sessionID, session.ThetaCurrent)

	return &models.CloseSessionResponse{
		SessionID:         sessionID,
		Status:            "COMPLETED",
		ThetaStart:        session.ThetaStart,
		ThetaFinal:        session.ThetaCurrent,
		MasteryScore:      masteryScore,
		Accuracy:          accuracy,
		TotalQuestions:    totalQuestions,
		TotalSolved:       totalSolved,
		PerTopicBreakdown: perTopicBreakdown,
	}, nil
}

// GetSession retrieves the complete state of an active or historical practice session.
func (ps *PracticeService) GetSession(ctx context.Context, userID, sessionID string) (*models.GetSessionResponse, error) {
	session, err := ps.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("practice_service: get session: %w", err)
	}

	if session.UserID != userID {
		return nil, fmt.Errorf("practice_service: unauthorized session access")
	}

	scope, _ := session.Scope()
	responses, _ := session.Responses()

	var currPayload *models.ProblemPayload
	if session.CurrentProblemID != nil {
		if p, err := ps.problemRepo.GetByID(ctx, *session.CurrentProblemID); err == nil && p != nil {
			payload := models.ToProblemPayload(p)
			currPayload = &payload
		}
	}

	return &models.GetSessionResponse{
		SessionID:      session.ID,
		UserID:         session.UserID,
		Mode:           session.Mode,
		Status:         session.Status,
		Scope:          scope,
		ThetaStart:     session.ThetaStart,
		ThetaCurrent:   session.ThetaCurrent,
		QuestionCount:  session.QuestionCount,
		CurrentProblem: currPayload,
		Responses:      responses,
		CreatedAt:      session.CreatedAt,
		UpdatedAt:      session.UpdatedAt,
	}, nil
}

// GetSimilar returns semantically related problems using vector embedding similarity search.
func (ps *PracticeService) GetSimilar(ctx context.Context, req models.SimilarProblemsRequest) ([]models.ProblemPayload, error) {
	prob, err := ps.problemRepo.GetByID(ctx, req.ProblemID)
	if err != nil {
		return nil, fmt.Errorf("practice_service: get source problem: %w", err)
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 5
	}

	similar, err := ps.problemRepo.FindSimilar(ctx, prob.Embedding, prob.Topic, limit+1)
	if err != nil {
		return nil, fmt.Errorf("practice_service: find similar: %w", err)
	}

	results := make([]models.Problem, 0, len(similar))
	for _, s := range similar {
		if s.ID != prob.ID {
			results = append(results, s)
			if len(results) >= limit {
				break
			}
		}
	}

	return models.ToProblemPayloads(results), nil
}

// GetTopics returns all curriculum topics paired with the user's current mastery scores and psychometrics.
func (ps *PracticeService) GetTopics(ctx context.Context, userID string) ([]models.TopicProgress, error) {
	userStats, _ := ps.statsRepo.GetByUser(ctx, userID)
	statsByTopic := make(map[string]models.TopicStats, len(userStats))
	for _, s := range userStats {
		statsByTopic[s.Topic] = s
	}

	allTopics := ps.graphSvc.graph.GetAllTopics()
	progressList := make([]models.TopicProgress, 0, len(allTopics))

	for _, node := range allTopics {
		stat, found := statsByTopic[node.ID]
		if found {
			progressList = append(progressList, models.TopicProgress{
				Topic:        node.ID,
				MasteryScore: stat.MasteryScore,
				Theta:        stat.Theta,
				GlickoRating: stat.GlickoRating,
				AttemptCount: stat.AttemptCount,
				CorrectCount: stat.CorrectCount,
			})
		} else {
			progressList = append(progressList, models.TopicProgress{
				Topic:        node.ID,
				MasteryScore: 0.0,
				Theta:        1500.0,
				GlickoRating: 1500.0,
				AttemptCount: 0,
				CorrectCount: 0,
			})
		}
	}

	return progressList, nil
}

// SearchProblems filters problems by query, topic, source, and difficulty.
func (ps *PracticeService) SearchProblems(ctx context.Context, req models.ProblemSearchRequest) (*models.ProblemSearchResponse, error) {
	problems, total, err := ps.problemRepo.Search(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("practice_service: search: %w", err)
	}

	return &models.ProblemSearchResponse{
		Total:    total,
		Problems: models.ToProblemPayloads(problems),
	}, nil
}

func marshalJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
