package services

import (
	"context"
	"fmt"
	"time"

	"velocity/internal/models"
	"velocity/internal/repository"
)

func (s *LearnService) GetChapters(ctx context.Context, userID string) ([]models.GraphNode, error) {
	ls, err := s.statsRepo.Get(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("learn: fetch stats: %w", err)
	}

	userChapters, err := ls.Chapters()
	if err != nil {
		return nil, fmt.Errorf("learn: parse chapters: %w", err)
	}

	chapterInfo, err := s.questionRepo.GetChapterInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("learn: get chapter info: %w", err)
	}

	nodes := make([]models.GraphNode, 0, len(chapterInfo))
	for _, ci := range chapterInfo {
		stats, hasPrior := userChapters[ci.Chapter]

		mastery := 0.0
		glickoRD := 350.0
		theta := 1300.0
		lastSeen := ""

		if hasPrior {
			theta = stats.Theta
			glickoRD = stats.GlickoRD
			mastery = stats.MasteryScore
			if !stats.LastPracticedAt.IsZero() {
				lastSeen = stats.LastPracticedAt.Format(time.RFC3339)
			}
		}

		nodes = append(nodes, models.GraphNode{
			ID:             ci.Chapter,
			Chapter:        repository.SlugToDisplayName(ci.Chapter),
			Subject:        ci.Subject,
			Group:          ci.ChapterGroup,
			MasteryScore:   mastery,
			GlickoRD:       glickoRD,
			Theta:          theta,
			LastSeen:       lastSeen,
			TotalQuestions: ci.TotalQuestions,
			VeryEasyCount:  ci.VeryEasyCount,
			EasyCount:      ci.EasyCount,
			MediumCount:    ci.MediumCount,
			HardCount:      ci.HardCount,
			VeryHardCount:  ci.VeryHardCount,
		})
	}

	return nodes, nil
}

func (s *LearnService) GetLearnPage(ctx context.Context, userID string) (*models.LearnPageResponse, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("learn: get user: %w", err)
	}

	settings, err := user.Settings()
	if err != nil {
		return nil, fmt.Errorf("learn: parse settings: %w", err)
	}

	ls, err := s.statsRepo.Get(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("learn: fetch stats: %w", err)
	}

	userChapters, err := ls.Chapters()
	if err != nil {
		return nil, fmt.Errorf("learn: parse chapters: %w", err)
	}

	chapterInfo, err := s.questionRepo.GetChapterInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("learn: get chapter info: %w", err)
	}

	availableExamTypes, err := s.questionRepo.GetAvailableExamTypes(ctx)
	if err != nil {
		return nil, fmt.Errorf("learn: get exam types: %w", err)
	}

	chapters := make([]models.LearnChapterNode, 0, len(chapterInfo))
	for _, ci := range chapterInfo {
		stats, hasPrior := userChapters[ci.Chapter]

		mastery := 0.0
		glickoRD := 350.0
		theta := 1300.0
		lastPracticed := ""
		totalAttempts := 0
		correctAttempts := 0

		if hasPrior {
			theta = stats.Theta
			glickoRD = stats.GlickoRD
			mastery = stats.MasteryScore
			totalAttempts = stats.TotalAttempts
			correctAttempts = stats.CorrectAttempts
			if !stats.LastPracticedAt.IsZero() {
				lastPracticed = stats.LastPracticedAt.Format(time.RFC3339)
			}
		}

		chapters = append(chapters, models.LearnChapterNode{
			ID:              ci.Chapter,
			Chapter:         repository.SlugToDisplayName(ci.Chapter),
			Subject:         ci.Subject,
			Group:           ci.ChapterGroup,
			MasteryScore:    mastery,
			GlickoRD:        glickoRD,
			Theta:           theta,
			LastPracticedAt: lastPracticed,
			TotalAttempts:   totalAttempts,
			CorrectAttempts: correctAttempts,
			TotalQuestions:  ci.TotalQuestions,
			VeryEasyCount:   ci.VeryEasyCount,
			EasyCount:       ci.EasyCount,
			MediumCount:     ci.MediumCount,
			HardCount:       ci.HardCount,
			VeryHardCount:   ci.VeryHardCount,
			ExamTypes:       ci.ExamTypes,
		})
	}

	activeSess, err := s.sessionRepo.GetActiveByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("learn: fetch active session: %w", err)
	}

	var activeSessionInfo *models.LastSessionInfo
	if activeSess != nil {
		activeSessionInfo = s.buildLastSessionInfo(activeSess)
	}

	lastSession, err := s.sessionRepo.GetLastCompletedByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("learn: fetch last session: %w", err)
	}

	var lastSessionInfo *models.LastSessionInfo
	if lastSession != nil {
		lastSessionInfo = s.buildLastSessionInfo(lastSession)
	}

	// Build subjects_per_exam from chapter info: for each exam type, collect unique subjects
	subjectsPerExam := make(map[string][]string)
	seen := make(map[string]map[string]bool) // exam_type → subject → seen
	for _, ci := range chapterInfo {
		for _, et := range ci.ExamTypes {
			if seen[et] == nil {
				seen[et] = make(map[string]bool)
			}
			if !seen[et][ci.Subject] {
				seen[et][ci.Subject] = true
				subjectsPerExam[et] = append(subjectsPerExam[et], ci.Subject)
			}
		}
	}

	return &models.LearnPageResponse{
		Chapters:           chapters,
		ActiveSession:      activeSessionInfo,
		LastSession:        lastSessionInfo,
		DefaultExamType:    settings.DefaultExamType,
		AvailableExamTypes: availableExamTypes,
		SubjectsPerExam:    subjectsPerExam,
	}, nil
}

func (s *LearnService) buildLastSessionInfo(session *models.LearnSession) *models.LastSessionInfo {
	responses, err := session.Responses()
	if err != nil {
		return nil
	}
	correct, total := 0, 0
	for _, r := range responses {
		if r.Skipped {
			continue
		}
		total++
		if r.IsCorrect {
			correct++
		}
	}
	accuracy := 0.0
	if total > 0 {
		accuracy = float64(correct) / float64(total)
	}
	return &models.LastSessionInfo{
		SessionID:      session.ID,
		Chapter:        session.Chapter,
		Mode:           session.Mode,
		TotalQuestions: total,
		CorrectCount:   correct,
		Accuracy:       accuracy,
		CompletedAt:    session.UpdatedAt.Format(time.RFC3339),
	}
}
