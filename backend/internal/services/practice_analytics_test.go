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

func TestComputeSessionStreaks(t *testing.T) {
	responses := []models.SessionResponse{
		{ProblemID: "p1", IsCorrect: true},
		{ProblemID: "p2", IsCorrect: true},
		{ProblemID: "p3", IsCorrect: true},
	}

	correct, wrong := ComputeSessionStreaks(responses)
	if correct != 3 || wrong != 0 {
		t.Errorf("expected 3 correct, 0 wrong; got %d, %d", correct, wrong)
	}
}

func TestUpdateLearningDNA(t *testing.T) {
	existingDNA := models.DefaultDNA()
	existingDNA.StreakRecord = 2

	problemMap := map[string]models.Problem{
		"p1": {ID: "p1", Topic: "arrays", Source: "leetcode", GlickoRating: 1400.0},
		"p2": {ID: "p2", Topic: "arrays", Source: "leetcode", GlickoRating: 1450.0},
		"p3": {ID: "p3", Topic: "arrays", Source: "leetcode", GlickoRating: 1500.0},
		"p4": {ID: "p4", Topic: "dp", Source: "codeforces", GlickoRating: 1600.0},
	}

	responses := []models.SessionResponse{
		{ProblemID: "p1", IsCorrect: true, TimeTakenMs: 60000, ThetaBefore: 1700},
		{ProblemID: "p2", IsCorrect: true, TimeTakenMs: 70000, ThetaBefore: 1700},
		{ProblemID: "p3", IsCorrect: true, TimeTakenMs: 80000, ThetaBefore: 1700},
		{ProblemID: "p4", IsCorrect: false, TimeTakenMs: 90000, ThetaBefore: 1700},
	}

	duration := 15 * time.Minute
	newDNA := UpdateLearningDNA(existingDNA, responses, duration, problemMap)

	if newDNA.TotalSessions != 1 {
		t.Errorf("expected TotalSessions 1, got %d", newDNA.TotalSessions)
	}
	if newDNA.TotalProblemsSolved != 3 {
		t.Errorf("expected TotalProblemsSolved 3, got %d", newDNA.TotalProblemsSolved)
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
}
