package services

import (
	"math"
	"time"

	"transverse/internal/models"
)

// CalculateMasteryScore normalizes an IRT ability estimate (theta) onto a [0, 100] scale.
// Baseline 1300 corresponds to 0%, 2800+ corresponds to 100%.
func CalculateMasteryScore(theta float64) float64 {
	minTheta := 1300.0
	maxTheta := 2800.0
	if theta <= minTheta {
		return 0.0
	}
	if theta >= maxTheta {
		return 100.0
	}
	score := ((theta - minTheta) / (maxTheta - minTheta)) * 100.0
	return math.Round(score*10) / 10
}

// ComputeSessionStreaks scans a sequence of session responses and returns
// the consecutive correct and consecutive wrong streak counts at the tail of the session.
func ComputeSessionStreaks(responses []models.SessionResponse) (consecutiveCorrect int, consecutiveWrong int) {
	if len(responses) == 0 {
		return 0, 0
	}

	for i := len(responses) - 1; i >= 0; i-- {
		resp := responses[i]
		if resp.IsCorrect && !resp.Skipped {
			if consecutiveWrong > 0 {
				break
			}
			consecutiveCorrect++
		} else {
			if consecutiveCorrect > 0 {
				break
			}
			consecutiveWrong++
		}
	}

	return consecutiveCorrect, consecutiveWrong
}

// UpdateLearningDNA applies exponential moving average (EMA) smoothing and updates
// learner behavioral metrics following session completion.
func UpdateLearningDNA(
	current models.LearningDNA,
	sessionResponses []models.SessionResponse,
	sessionDuration time.Duration,
	problemMap map[string]models.Problem,
) models.LearningDNA {
	if len(sessionResponses) == 0 {
		return current
	}

	alpha := 0.15 // EMA smoothing factor

	totalInSession := len(sessionResponses)
	correctInSession := 0
	var totalTimeInSession int64
	platformCounts := make(map[string]int)
	topicSessionCorrect := make(map[string]int)
	topicSessionTotal := make(map[string]int)
	carelessWrong := 0
	carelessTotal := 0

	streak := 0
	maxSessionStreak := 0

	for _, resp := range sessionResponses {
		totalTimeInSession += resp.TimeTakenMs
		
		// Get problem metadata if available
		if p, ok := problemMap[resp.ProblemID]; ok {
			if p.Source != "" {
				platformCounts[p.Source]++
			}
			if p.Topic != "" {
				topicSessionTotal[p.Topic]++
				if resp.IsCorrect && !resp.Skipped {
					topicSessionCorrect[p.Topic]++
				}
			}
			// Carelessness: wrong on problems where difficulty <= theta - 200
			if p.GlickoRating <= resp.ThetaBefore-200 {
				carelessTotal++
				if !resp.IsCorrect {
					carelessWrong++
				}
			}
		}

		if resp.IsCorrect && !resp.Skipped {
			correctInSession++
			streak++
			if streak > maxSessionStreak {
				maxSessionStreak = streak
			}
		} else {
			streak = 0
		}
	}

	sessionAccuracy := 0.0
	if totalInSession > 0 {
		sessionAccuracy = float64(correctInSession) / float64(totalInSession)
	}

	meanSessionTimeMs := int64(0)
	if totalInSession > 0 {
		meanSessionTimeMs = totalTimeInSession / int64(totalInSession)
	}

	// Update rolling metrics with EMA
	if current.TotalSessions == 0 {
		current.AvgAccuracy = sessionAccuracy
		current.AvgTimeTakenMs = meanSessionTimeMs
		current.AvgSessionLength = float64(totalInSession)
	} else {
		current.AvgAccuracy = (alpha * sessionAccuracy) + ((1.0 - alpha) * current.AvgAccuracy)
		current.AvgTimeTakenMs = int64((alpha * float64(meanSessionTimeMs)) + ((1.0 - alpha) * float64(current.AvgTimeTakenMs)))
		current.AvgSessionLength = (alpha * float64(totalInSession)) + ((1.0 - alpha) * current.AvgSessionLength)
	}

	// Solve velocity: problems per hour
	hours := sessionDuration.Hours()
	if hours > 0.01 {
		sessionVelocity := float64(correctInSession) / hours
		if current.TotalSessions == 0 {
			current.AvgSolveVelocity = sessionVelocity
		} else {
			current.AvgSolveVelocity = (alpha * sessionVelocity) + ((1.0 - alpha) * current.AvgSolveVelocity)
		}
	}

	// Carelessness index
	if carelessTotal > 0 {
		sessionCarelessness := float64(carelessWrong) / float64(carelessTotal)
		if current.TotalSessions == 0 {
			current.CarelessnessIndex = sessionCarelessness
		} else {
			current.CarelessnessIndex = (alpha * sessionCarelessness) + ((1.0 - alpha) * current.CarelessnessIndex)
		}
	}

	// Cumulative counters
	current.TotalSessions++
	current.TotalProblemsSolved += correctInSession

	// Streak record
	if maxSessionStreak > current.StreakRecord {
		current.StreakRecord = maxSessionStreak
	}

	// Update topic bias
	if current.TopicBias == nil {
		current.TopicBias = make(map[string]float64)
	}

	for topic, count := range topicSessionTotal {
		if count > 0 {
			topicAcc := float64(topicSessionCorrect[topic]) / float64(count)
			delta := topicAcc - current.AvgAccuracy
			existingBias := current.TopicBias[topic]
			current.TopicBias[topic] = (alpha * delta) + ((1.0 - alpha) * existingBias)
		}
	}

	// Preferred platform
	topPlatform := current.PreferredPlatform
	maxCount := 0
	for p, count := range platformCounts {
		if count > maxCount {
			maxCount = count
			topPlatform = p
		}
	}
	if topPlatform != "" {
		current.PreferredPlatform = topPlatform
	}

	return current
}

// ComputeAvgOpponentRating calculates the mean Glicko rating of all attempted problems.
func ComputeAvgOpponentRating(responses []models.SessionResponse, problemMap map[string]models.Problem) float64 {
	if len(responses) == 0 {
		return 1500.0
	}
	var sum float64
	var count int
	for _, r := range responses {
		if p, ok := problemMap[r.ProblemID]; ok && p.GlickoRating > 0 {
			sum += p.GlickoRating
			count++
		}
	}
	if count == 0 {
		return 1500.0
	}
	return sum / float64(count)
}
