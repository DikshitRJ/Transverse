package services

import (
	"math"
	"testing"
	"time"

	"github.com/pgvector/pgvector-go"
	"transverse/internal/models"
)

func TestComputeEffectiveTheta(t *testing.T) {
	state := &ScState{
		ThetaCurrent:       1500.0,
		Topic:              "arrays-hashing",
		TopicBias:          0.2, // +40
		ConsecutiveCorrect: 3,   // +45 momentum
		ConsecutiveWrong:   0,
		QuestionCount:      5,
		AvgSessionLength:   10, // 50% -> normal phase (no adjustment)
	}

	eff, momentum := ComputeEffectiveTheta(state)

	// Topic bias = 0.2 * 200 = +40
	// Momentum = 3 * 15 = +45
	// Session phase = 5/10 = 0.5 (normal -> 0)
	// eff = 1500 + 40 + 45 = 1585.0
	expectedEff := 1585.0
	expectedMom := 45.0

	if math.Abs(eff-expectedEff) > 1e-4 {
		t.Errorf("ComputeEffectiveTheta() eff = %f; want %f", eff, expectedEff)
	}
	if math.Abs(momentum-expectedMom) > 1e-4 {
		t.Errorf("ComputeEffectiveTheta() momentum = %f; want %f", momentum, expectedMom)
	}
}

func TestComputeEffectiveTheta_WarmupAndCooldown(t *testing.T) {
	// Warmup: question count 1, avg session length 10 -> pct = 0.1 < 0.2 -> -30
	warmupState := &ScState{
		ThetaCurrent:     1500.0,
		QuestionCount:    1,
		AvgSessionLength: 10,
	}
	effWarmup, _ := ComputeEffectiveTheta(warmupState)
	if math.Abs(effWarmup-1470.0) > 1e-4 {
		t.Errorf("warmup eff = %f; want 1470.0", effWarmup)
	}

	// Cooldown: question count 8, avg session length 10 -> pct = 0.8 >= 0.7 -> -20
	cooldownState := &ScState{
		ThetaCurrent:     1500.0,
		QuestionCount:    8,
		AvgSessionLength: 10,
	}
	effCooldown, _ := ComputeEffectiveTheta(cooldownState)
	if math.Abs(effCooldown-1480.0) > 1e-4 {
		t.Errorf("cooldown eff = %f; want 1480.0", effCooldown)
	}
}

func TestTuneWeights(t *testing.T) {
	initial := WeightSet{
		DifficultyFit:       0.50,
		ConceptSimilarity:   0.15,
		TopicProgression:    0.10,
		NoveltyFactor:       0.10,
		ImmediateReinforce:  0.05,
		PlatformDiversity:   0.05,
		CarelessnessPenalty: 0.05,
	}

	tuned := tuneWeights(initial, 0.4)

	// Check weights sum to 1.0
	sum := tuned.DifficultyFit + tuned.ConceptSimilarity + tuned.TopicProgression +
		tuned.NoveltyFactor + tuned.ImmediateReinforce + tuned.PlatformDiversity + tuned.CarelessnessPenalty

	if math.Abs(sum-1.0) > 1e-5 {
		t.Errorf("tuned weights sum to %f; want 1.0", sum)
	}

	// DifficultyFit should increase and CarelessnessPenalty should decrease
	if tuned.DifficultyFit <= initial.DifficultyFit {
		t.Errorf("expected DifficultyFit to increase after carelessness tuning, got %f", tuned.DifficultyFit)
	}
}

func TestScoreCandidate_PlatformDiversityAndNovelty(t *testing.T) {
	v1 := pgvector.NewVector([]float32{1.0, 0.0, 0.0})
	v2 := pgvector.NewVector([]float32{1.0, 0.0, 0.0})

	cur := &models.Problem{
		ID:           "prob_1",
		Topic:        "arrays-hashing",
		GlickoRating: 1500.0,
		Embedding:    v1,
		Source:       "leetcode",
	}

	cand := models.Problem{
		ID:           "prob_2",
		Topic:        "arrays-hashing",
		GlickoRating: 1500.0,
		Embedding:    v2,
		Source:       "codeforces",
	}

	state := &ScState{
		ThetaCurrent: 1500.0,
		Topic:        "arrays-hashing",
		AttemptCounts: map[string]int{
			"prob_2": 0, // new problem
		},
		SessionSourceCounts: map[string]int{
			"leetcode":   4,
			"codeforces": 0, // underrepresented
		},
		CarelessnessIndex: 0.0,
	}

	weights := correctWeights
	comp := ScoreCandidate(cand, cur, 1500.0, state, weights)

	if comp.DifficultyFit != 1.0 {
		t.Errorf("DifficultyFit = %f; want 1.0", comp.DifficultyFit)
	}
	if comp.ConceptSimilarity != 1.0 {
		t.Errorf("ConceptSimilarity = %f; want 1.0", comp.ConceptSimilarity)
	}
	if comp.TopicProgression != 0.3 {
		t.Errorf("TopicProgression = %f; want 0.3", comp.TopicProgression)
	}
	if comp.NoveltyFactor != 1.0 {
		t.Errorf("NoveltyFactor = %f; want 1.0", comp.NoveltyFactor)
	}
	if comp.PlatformDiversity != 1.0 {
		t.Errorf("PlatformDiversity = %f; want 1.0", comp.PlatformDiversity)
	}
	if comp.Total <= 0.0 || comp.Total > 1.0 {
		t.Errorf("Total = %f out of [0,1] range", comp.Total)
	}
}

func TestAfterCorrectFilter(t *testing.T) {
	current := &models.Problem{
		ID:           "p_mid",
		GlickoRating: 1500.0,
	}

	candidates := []models.Problem{
		{ID: "p_low", GlickoRating: 1200.0},
		{ID: "p_mid", GlickoRating: 1500.0},
		{ID: "p_high", GlickoRating: 1700.0},
	}

	filtered := afterCorrectFilter(candidates, current)
	if len(filtered) != 1 || filtered[0].ID != "p_high" {
		t.Fatalf("expected [p_high], got %+v", filtered)
	}

	// Fallback when none harder
	maxCandidate := &models.Problem{ID: "p_max", GlickoRating: 2000.0}
	filteredFallback := afterCorrectFilter(candidates, maxCandidate)
	if len(filteredFallback) != 3 {
		t.Errorf("expected fallback to return 3 candidates, got %d", len(filteredFallback))
	}
}

func TestAfterWrongFilter(t *testing.T) {
	current := &models.Problem{
		ID:    "p1",
		Topic: "binary-search",
	}

	candidates := []models.Problem{
		{ID: "p1", Topic: "binary-search"},
		{ID: "p2", Topic: "binary-search"},
		{ID: "p3", Topic: "dynamic-programming"},
	}

	filtered := afterWrongFilter(candidates, current)
	if len(filtered) != 1 || filtered[0].ID != "p2" {
		t.Fatalf("expected [p2], got %+v", filtered)
	}
}

func TestPickBestProblem_ColdStart(t *testing.T) {
	candidates := []models.Problem{
		{ID: "p1", GlickoRating: 1000.0, Topic: "arrays-hashing"},
		{ID: "p2", GlickoRating: 1300.0, Topic: "arrays-hashing"},
		{ID: "p3", GlickoRating: 1800.0, Topic: "arrays-hashing"},
	}

	state := &ScState{
		ThetaCurrent: 1300.0,
		Topic:        "arrays-hashing",
	}

	res := PickBestProblem(candidates, state, nil, false)
	if res == nil {
		t.Fatal("expected non-nil PickResult")
	}
	if res.Problem.ID != "p2" {
		t.Errorf("expected cold start to pick closest difficulty p2 (1300), got %s", res.Problem.ID)
	}
}

func TestPickAfterSkip(t *testing.T) {
	candidates := []models.Problem{
		{ID: "p1", GlickoRating: 1200.0, Topic: "two-pointers"},
		{ID: "p2", GlickoRating: 1500.0, Topic: "two-pointers"},
	}

	skipped := &models.Problem{
		ID:    "p_skip",
		Topic: "two-pointers",
	}

	state := &ScState{
		ThetaCurrent:       1400.0,
		ConsecutiveCorrect: 4,
	}

	res := PickAfterSkip(candidates, state, skipped)
	if res == nil {
		t.Fatal("expected non-nil PickResult after skip")
	}
	if res.Momentum != 0 {
		t.Errorf("expected streak momentum to be reset to 0 after skip, got %f", res.Momentum)
	}
}

func TestComputeStreaks(t *testing.T) {
	now := time.Now()
	responses := []models.SessionResponse{
		{ProblemID: "1", IsCorrect: false, SubmittedAt: now.Add(-3 * time.Minute)},
		{ProblemID: "2", IsCorrect: true, SubmittedAt: now.Add(-2 * time.Minute)},
		{ProblemID: "3", IsCorrect: true, SubmittedAt: now.Add(-1 * time.Minute)},
	}

	// Latest answer is correct -> streak should be 3
	c, w := computeStreaks(responses, true)
	if c != 3 || w != 0 {
		t.Errorf("computeStreaks(correct) = (%d, %d); want (3, 0)", c, w)
	}

	// Latest answer is wrong -> streak should be 1 wrong
	c2, w2 := computeStreaks(responses, false)
	if c2 != 0 || w2 != 1 {
		t.Errorf("computeStreaks(wrong) = (%d, %d); want (0, 1)", c2, w2)
	}
}
