package services

import (
	"math"
	"time"

	"velocity/internal/models"
	"velocity/internal/repository"
)

var goTimeNow = time.Now // overridable in tests

func cosineSimilarity(a, b []float32) float64 {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

func computeDynamicRD(attemptCount int) float32 {
	switch {
	case attemptCount <= 5:
		return 350.0
	case attemptCount <= 20:
		return 200.0
	case attemptCount <= 50:
		return 100.0
	default:
		return 50.0
	}
}

func BuildTrend(rows []repository.DailyTrendRow, earliest time.Time, days int) []models.DailyTrendEntry {
	trendMap := make(map[string]models.DailyTrendEntry)
	for _, r := range rows {
		key := r.Date.Format("2006-01-02")
		trendMap[key] = models.DailyTrendEntry{
			Date:    key,
			Correct: r.Correct,
			Wrong:   r.Wrong,
			Total:   r.Total,
		}
	}

	// For "all" range (days <= 0), compute actual span from earliest to today.
	var start time.Time
	if days <= 0 {
		start = earliest
		days = int(time.Now().Sub(earliest).Hours()/24) + 1
		if days < 1 {
			days = 1
		}
	} else {
		start = time.Now().AddDate(0, 0, -days+1)
	}

	var result []models.DailyTrendEntry
	for i := 0; i < days; i++ {
		date := start.AddDate(0, 0, i)
		key := date.Format("2006-01-02")
		if entry, ok := trendMap[key]; ok {
			result = append(result, entry)
		} else {
			result = append(result, models.DailyTrendEntry{
				Date: key,
			})
		}
	}
	return result
}

// computeTrendChange computes the change in correct count and accuracy between
// the current period and the previous period.
func computeTrendChange(currentTrend []models.DailyTrendEntry, prevRows []repository.DailyTrendRow, periodDays int) *models.TrendChange {
	// Aggregate current period
	var correctThis, totalThis int
	for _, e := range currentTrend {
		correctThis += e.Correct
		totalThis += e.Total
	}

	// Aggregate previous period
	var correctPrev, totalPrev int
	for _, r := range prevRows {
		correctPrev += r.Correct
		totalPrev += r.Total
	}

	if totalPrev == 0 {
		return nil
	}

	correctChange := (float64(correctThis-correctPrev) / float64(correctPrev)) * 100.0

	accThis := 0.0
	if totalThis > 0 {
		accThis = float64(correctThis) / float64(totalThis)
	}
	accPrev := float64(correctPrev) / float64(totalPrev)
	accuracyChange := (accThis - accPrev) * 100.0 // percentage points

	return &models.TrendChange{
		CorrectChange:   math.Round(correctChange*10) / 10,
		AccuracyChange:  math.Round(accuracyChange*10) / 10,
		CorrectThis:     correctThis,
		CorrectPrevious: correctPrev,
		TotalThis:       totalThis,
		TotalPrevious:   totalPrev,
	}
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// computeStreak calculates the current consecutive-day practice streak.
// A streak is the number of consecutive days (ending today or yesterday)
// in which the user practiced at least one session across any chapter.
// If the latest practice was more than 1 day ago, the streak is broken (returns 0).
func computeStreak(chapters map[string]models.ChapterStats) int {
	if len(chapters) == 0 {
		return 0
	}

	// Collect all unique practice dates (date only, no time).
	dates := make(map[string]bool)
	for _, cs := range chapters {
		if !cs.LastPracticedAt.IsZero() {
			dates[cs.LastPracticedAt.Format("2006-01-02")] = true
		}
	}
	if len(dates) == 0 {
		return 0
	}

	now := goTimeNow().Truncate(24 * time.Hour)

	// Find the latest practice date.
	var latest time.Time
	for d := range dates {
		t, err := time.Parse("2006-01-02", d)
		if err != nil {
			continue
		}
		if t.After(latest) {
			latest = t
		}
	}
	if latest.IsZero() {
		return 0
	}

	// Streak is broken if the latest practice was more than 1 day before today.
	if now.Sub(latest) > 24*time.Hour {
		return 0
	}

	// Count consecutive days backward from the latest practice date.
	streak := 0
	for i := 0; ; i++ {
		date := latest.AddDate(0, 0, -i).Format("2006-01-02")
		if dates[date] {
			streak++
		} else {
			break
		}
	}
	return streak
}
