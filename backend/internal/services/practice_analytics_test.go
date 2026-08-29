package services

import (
	"testing"
	"time"

	"transverse/internal/models"
)

func TestComputeAvgOpponentRating(t *testing.T) {
	problemMap := map[string]models.Problem{
		"p1": {ID: "p1", GlickoRating: 1400.0},
		"p2": {ID: "p2", GlickoRating: 1600.0},
		"p3": {ID: "p3", GlickoRating: 1800.0},
	}

	responses := []models.SessionResponse{
		{ProblemID: "p1"},
		{ProblemID: "p2"},
		{ProblemID: "p3"},
	}

	avg := ComputeAvgOpponentRating(responses, problemMap)
	expected := (1400.0 + 1600.0 + 1800.0) / 3.0
	if avg != expected {
		t.Errorf("expected avg %f, got %f", expected, avg)
	}

	emptyAvg := ComputeAvgOpponentRating([]models.SessionResponse{}, problemMap)
	if emptyAvg != 1500.0 {
		t.Errorf("expected default 1500.0 for empty responses, got %f", emptyAvg)
	}
}

func TestComputeCarelessness(t *testing.T) {
	problemMap := map[string]models.Problem{
		"easy_wrong": {ID: "easy_wrong", GlickoRating: 1200.0},
		"hard_wrong": {ID: "hard_wrong", GlickoRating: 1700.0},
		"easy_right": {ID: "easy_right", GlickoRating: 1200.0},
	}

	userTheta := 1500.0 // threshold is userTheta - 200 = 1300.0

	responses := []models.SessionResponse{
		{ProblemID: "easy_wrong", IsCorrect: false},
		{ProblemID: "hard_wrong", IsCorrect: false},
		{ProblemID: "easy_right", IsCorrect: true},
	}

	carelessness := computeCarelessness(responses, problemMap, userTheta)
	// 2 wrong answers total: 1 easy_wrong (< 1300), 1 hard_wrong (>= 1300) -> 1/2 = 0.5
	if carelessness != 0.5 {
		t.Errorf("expected carelessness 0.5, got %f", carelessness)
	}
}

func TestRecomputeDNA(t *testing.T) {
	existingDNA := models.DefaultDNA()
	existingDNA.StreakRecord = 2

	problems := []models.Problem{
		{ID: "p1", Topic: "arrays", Source: "leetcode", GlickoRating: 1400.0},
		{ID: "p2", Topic: "arrays", Source: "leetcode", GlickoRating: 1450.0},
		{ID: "p3", Topic: "arrays", Source: "leetcode", GlickoRating: 1500.0},
		{ID: "p4", Topic: "dp", Source: "codeforces", GlickoRating: 1600.0},
	}

	responses := []models.SessionResponse{
		{ProblemID: "p1", IsCorrect: true, TimeTakenMs: 60000},
		{ProblemID: "p2", IsCorrect: true, TimeTakenMs: 70000},
		{ProblemID: "p3", IsCorrect: true, TimeTakenMs: 80000},
		{ProblemID: "p4", IsCorrect: false, TimeTakenMs: 90000},
	}

	durationMs := int64(300000)
	thetaFinal := 1550.0

	newDNA := RecomputeDNA(existingDNA, responses, problems, durationMs, thetaFinal)

	if newDNA.TotalSessions != 1 {
		t.Errorf("expected TotalSessions 1, got %d", newDNA.TotalSessions)
	}
	if newDNA.TotalProblemsSolved != 4 {
		t.Errorf("expected TotalProblemsSolved 4, got %d", newDNA.TotalProblemsSolved)
	}
	if newDNA.AvgAccuracy != 0.75 { // 3 correct out of 4
		t.Errorf("expected AvgAccuracy 0.75, got %f", newDNA.AvgAccuracy)
	}
	if newDNA.StreakRecord != 3 {
		t.Errorf("expected StreakRecord 3, got %d", newDNA.StreakRecord)
	}
	if newDNA.PreferredPlatform != "leetcode" {
		t.Errorf("expected PreferredPlatform leetcode, got %s", newDNA.PreferredPlatform)
	}
	// "arrays" has 3 problems (>= 3), so TopicBias["arrays"] should be updated
	if _, ok := newDNA.TopicBias["arrays"]; !ok {
		t.Errorf("expected TopicBias for arrays to be updated")
	}
	// "dp" has only 1 problem (< 3), so TopicBias["dp"] should NOT be updated
	if _, ok := newDNA.TopicBias["dp"]; ok {
		t.Errorf("did not expect TopicBias for dp since count < 3")
	}
}

func TestCacheKeys(t *testing.T) {
	if CacheKeyProblem("p123") != "problem:p123" {
		t.Errorf("unexpected problem key: %s", CacheKeyProblem("p123"))
	}
	if CacheKeySeenIDs("u123") != "seen:u123" {
		t.Errorf("unexpected seen key: %s", CacheKeySeenIDs("u123"))
	}
	if CacheKeyDNA("u123") != "dna:u123" {
		t.Errorf("unexpected dna key: %s", CacheKeyDNA("u123"))
	}
	if CacheKeyTopicStats("u123") != "topic_stats:u123" {
		t.Errorf("unexpected topic stats key: %s", CacheKeyTopicStats("u123"))
	}
}
