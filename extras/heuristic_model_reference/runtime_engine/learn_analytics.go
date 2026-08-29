package services

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"velocity/internal/models"
)

func (s *LearnService) recomputeDNA(ctx context.Context, userID string, sessionResponses []models.SessionResponse) (models.LearningDNA, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return models.LearningDNA{}, err
	}
	dna, err := user.DNA()
	if err != nil {
		return models.LearningDNA{}, err
	}
	totalQ := len(sessionResponses)
	if totalQ == 0 {
		return dna, nil
	}
	correctCount := 0
	var totalTimeMs int64
	for _, r := range sessionResponses {
		if r.IsCorrect {
			correctCount++
		}
		totalTimeMs += r.TimeTakenMs
	}
	accuracy := float64(correctCount) / float64(totalQ)
	avgTimeMs := totalTimeMs / int64(totalQ)
	n := float64(dna.TotalQuestionsSolved)
	if n == 0 {
		dna.AvgAccuracy = accuracy
		dna.AvgTimeTakenMs = avgTimeMs
	} else {
		dna.AvgAccuracy = (dna.AvgAccuracy*n + accuracy) / (n + float64(totalQ))
		dna.AvgTimeTakenMs = int64((float64(dna.AvgTimeTakenMs)*n + float64(totalTimeMs)) / (n + float64(totalQ)))
	}
	dna.TotalQuestionsSolved += totalQ
	dna.TotalSessions++
	if dna.AvgSessionLength == 0 {
		dna.AvgSessionLength = float64(totalQ)
	} else {
		dna.AvgSessionLength = (dna.AvgSessionLength*float64(dna.TotalSessions-1) + float64(totalQ)) / float64(dna.TotalSessions)
	}

	// ── Solve velocity (questions per hour) ──────────────────────────────
	if avgTimeMs > 0 {
		sessionVelocity := 3600000.0 / float64(avgTimeMs) // ms/q → q/hr
		if dna.AvgSolveVelocity == 0 {
			dna.AvgSolveVelocity = sessionVelocity
		} else {
			dna.AvgSolveVelocity = (dna.AvgSolveVelocity*n + sessionVelocity*float64(totalQ)) / (n + float64(totalQ))
		}
	}

	// ── Carelessness index (fraction of wrong answers faster than median) ─
	if totalQ > 0 {
		// Compute session median time
		times := make([]int64, 0, totalQ)
		for _, r := range sessionResponses {
			if r.TimeTakenMs > 0 {
				times = append(times, r.TimeTakenMs)
			}
		}
		if len(times) > 0 {
			sort.Slice(times, func(i, j int) bool { return times[i] < times[j] })
			median := times[len(times)/2]

			wrongCount := 0
			fastWrongCount := 0
			for _, r := range sessionResponses {
				if r.Skipped || r.IsCorrect {
					continue
				}
				wrongCount++
				if r.TimeTakenMs < median {
					fastWrongCount++
				}
			}
			if wrongCount > 0 {
				sessionCarelessness := float64(fastWrongCount) / float64(wrongCount)
				if dna.CarelessnessIndex == 0 {
					dna.CarelessnessIndex = sessionCarelessness
				} else {
					dna.CarelessnessIndex = (dna.CarelessnessIndex*n + sessionCarelessness*float64(totalQ)) / (n + float64(totalQ))
				}
			}
		}
	}

	// ── Peak performance hour (hour of day with best accuracy this session) ─
	hourCorrect := make(map[int]int)
	hourTotal := make(map[int]int)
	for _, r := range sessionResponses {
		if r.Skipped {
			continue
		}
		h := r.SubmittedAt.Hour()
		hourTotal[h]++
		if r.IsCorrect {
			hourCorrect[h]++
		}
	}
	bestHour := dna.PeakPerformanceHour // keep existing if session has no clear signal
	bestAcc := -1.0
	for h := 0; h < 24; h++ {
		tot := hourTotal[h]
		if tot < 2 {
			continue
		}
		acc := float64(hourCorrect[h]) / float64(tot)
		if acc > bestAcc {
			bestAcc = acc
			bestHour = h
		}
	}
	if bestAcc >= 0 {
		dna.PeakPerformanceHour = bestHour
	}

	subjectCorrect := make(map[string]float64)
	subjectTotal := make(map[string]float64)

	questionIDs := make([]string, len(sessionResponses))
	for i, r := range sessionResponses {
		questionIDs[i] = r.QuestionID
	}
	subjectsByID, err := s.loadQuestionSubjectsBatch(ctx, questionIDs)
	if err != nil {
		return dna, nil
	}

	for _, r := range sessionResponses {
		subj := subjectsByID[r.QuestionID]
		if subj == "" {
			continue
		}
		subjectTotal[subj]++
		if r.IsCorrect {
			subjectCorrect[subj]++
		}
	}

	if dna.SubjectBias == nil {
		dna.SubjectBias = map[string]float64{
			"physics":   0.0,
			"chemistry": 0.0,
			"maths":     0.0,
		}
	}

	const biasAlpha = 0.15
	for subj, total := range subjectTotal {
		if total < 3 {
			continue
		}
		sessionSubjAcc := subjectCorrect[subj] / total
		delta := sessionSubjAcc - dna.AvgAccuracy
		old := dna.SubjectBias[subj]
		dna.SubjectBias[subj] = old*(1-biasAlpha) + delta*biasAlpha
		dna.SubjectBias[subj] = math.Max(-0.5, math.Min(0.5, dna.SubjectBias[subj]))
	}

	return dna, nil
}

func (s *LearnService) GetSessionAnalysis(ctx context.Context, userID, sessionID string) (*models.SessionAnalysisResponse, error) {
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("learn: session not found: %w", err)
	}
	if session.UserID != userID {
		return nil, fmt.Errorf("learn: session belongs to different user")
	}

	responses, err := session.Responses()
	if err != nil {
		return nil, fmt.Errorf("learn: parse responses: %w", err)
	}

	totalQ := len(responses)
	correctCount := 0
	wrongCount := 0
	skipCount := 0
	var totalTimeMs int64
	fastestMs := int64(0)
	slowestMs := int64(0)

	analysisQuestions := make([]models.SessionQuestionAnalysis, 0, totalQ)
	var firstSubmitted, lastSubmitted time.Time

	for _, r := range responses {
		if r.Skipped {
			skipCount++
		} else if r.IsCorrect {
			correctCount++
		} else {
			wrongCount++
		}
		totalTimeMs += r.TimeTakenMs

		if r.TimeTakenMs > 0 {
			if fastestMs == 0 || r.TimeTakenMs < fastestMs {
				fastestMs = r.TimeTakenMs
			}
			if r.TimeTakenMs > slowestMs {
				slowestMs = r.TimeTakenMs
			}
		}

		if firstSubmitted.IsZero() || r.SubmittedAt.Before(firstSubmitted) {
			firstSubmitted = r.SubmittedAt
		}
		if r.SubmittedAt.After(lastSubmitted) {
			lastSubmitted = r.SubmittedAt
		}

		q, err := s.loadQuestionByID(ctx, r.QuestionID)
		if err != nil {
			continue
		}
		qp := models.ToQuestionPayload(q)
		s.EnrichAttemptCount(ctx, userID, &qp)

		analysisQuestions = append(analysisQuestions, models.SessionQuestionAnalysis{
			QuestionID:      r.QuestionID,
			Question:        qp,
			SelectedOptions: r.SelectedOptions,
			CorrectOptions:  q.CorrectOptions(),
			IsCorrect:       r.IsCorrect,
			Skipped:         r.Skipped,
			TimeTakenMs:     r.TimeTakenMs,
			ThetaBefore:     r.ThetaBefore,
			ThetaAfter:      r.ThetaAfter,
			SubmittedAt:     r.SubmittedAt,
		})
	}

	attemptedQ := totalQ - skipCount
	accuracy := 0.0
	if attemptedQ > 0 {
		accuracy = float64(correctCount) / float64(attemptedQ)
	}
	avgTimeMs := int64(0)
	if totalQ > 0 {
		avgTimeMs = totalTimeMs / int64(totalQ)
	}

	thetaFinal := float64(session.ThetaCurrent)
	masteryScore := ComputeMasteryScore(thetaFinal)

	completedAt := session.UpdatedAt
	durationMs := completedAt.Sub(session.CreatedAt).Milliseconds()

	resp := &models.SessionAnalysisResponse{
		SessionID:        session.ID,
		Mode:             session.Mode,
		Chapter:          session.Chapter,
		Status:           session.Status,
		ThetaStart:       float64(session.ThetaStart),
		ThetaFinal:       thetaFinal,
		MasteryScore:     masteryScore,
		CreatedAt:        session.CreatedAt,
		CompletedAt:      completedAt,
		BiometricEnabled: session.BiometricEnabled,

		TotalQuestions: totalQ,
		CorrectCount:   correctCount,
		WrongCount:     wrongCount,
		SkippedCount:   skipCount,
		Accuracy:       accuracy,

		AvgTimeTakenMs: avgTimeMs,
		TotalTimeMs:    totalTimeMs,
		FastestTimeMs:  fastestMs,
		SlowestTimeMs:  slowestMs,

		DurationMs: durationMs,

		Questions: analysisQuestions,
	}

	return resp, nil
}

func (s *LearnService) GetSessionHistory(ctx context.Context, userID string) (*models.SessionHistoryResponse, error) {
	sessions, err := s.sessionRepo.GetAllByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("learn: get session history: %w", err)
	}

	items := make([]models.SessionHistoryItem, 0, len(sessions))
	for _, sess := range sessions {
		responses, err := sess.Responses()
		if err != nil {
			continue
		}
		totalQ := len(responses)
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

		mastery := ComputeMasteryScore(float64(sess.ThetaCurrent))
		durationMs := sess.UpdatedAt.Sub(sess.CreatedAt).Milliseconds()

		items = append(items, models.SessionHistoryItem{
			SessionID:        sess.ID,
			Chapter:          sess.Chapter,
			Mode:             sess.Mode,
			Status:           sess.Status,
			TotalQuestions:   totalQ,
			CorrectCount:     correctCount,
			Accuracy:         accuracy,
			MasteryScore:     mastery,
			DurationMs:       durationMs,
			CreatedAt:        sess.CreatedAt,
			CompletedAt:      sess.UpdatedAt,
			BiometricEnabled: sess.BiometricEnabled,
		})
	}

	return &models.SessionHistoryResponse{Sessions: items}, nil
}
