package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"velocity/internal/models"
)

func (s *LearnService) loadQuestions(ctx context.Context, chapters []string) ([]models.Question, error) {
	questions, err := s.questionRepo.GetByChapters(ctx, chapters)
	if err != nil {
		return nil, err
	}
	applyDynamicRDToQuestions(questions)
	return questions, nil
}

func (s *LearnService) loadQuestionsFiltered(ctx context.Context, chapters, years, examTypes []string) ([]models.Question, error) {
	var questions []models.Question
	var err error
	if len(years) == 0 && len(examTypes) == 0 {
		questions, err = s.questionRepo.GetByChapters(ctx, chapters)
	} else {
		questions, err = s.questionRepo.GetByChaptersFiltered(ctx, chapters, years, examTypes)
	}
	if err != nil {
		return nil, err
	}
	applyDynamicRDToQuestions(questions)
	return questions, nil
}

func (s *LearnService) loadQuestionByID(ctx context.Context, id string) (*models.Question, error) {
	cacheKey := fmt.Sprintf("q:%s", id)
	var q models.Question
	if err := s.cache.Get(ctx, cacheKey, &q); err == nil {
		return &q, nil
	}

	qPtr, err := s.questionRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	qPtr.GlickoRD = computeDynamicRD(qPtr.AttemptCount)
	_ = s.cache.Set(ctx, cacheKey, qPtr, 24*time.Hour)
	return qPtr, nil
}

func applyDynamicRDToQuestions(questions []models.Question) {
	for i := range questions {
		questions[i].GlickoRD = computeDynamicRD(questions[i].AttemptCount)
	}
}

func (s *LearnService) loadSeenIDs(ctx context.Context, userID string) (map[string]int, error) {
	cacheKey := fmt.Sprintf("seen:%s", userID)
	var counts map[string]int
	if err := s.cache.Get(ctx, cacheKey, &counts); err == nil {
		return counts, nil
	}

	counts, err := s.questionStats.GetAllAttemptCounts(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("learn: load seen ids: %w", err)
	}
	_ = s.cache.Set(ctx, cacheKey, counts, 5*time.Minute)
	return counts, nil
}

func (s *LearnService) loadDNA(ctx context.Context, userID string) (models.LearningDNA, error) {
	cacheKey := fmt.Sprintf("dna:%s", userID)
	var dna models.LearningDNA
	if err := s.cache.Get(ctx, cacheKey, &dna); err == nil {
		return dna, nil
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return models.LearningDNA{}, err
	}
	dna, err = user.DNA()
	if err != nil {
		return models.LearningDNA{}, err
	}
	_ = s.cache.Set(ctx, cacheKey, dna, 60*time.Second)
	return dna, nil
}

func (s *LearnService) loadQuestionChapter(ctx context.Context, id string) (string, error) {
	cacheKey := fmt.Sprintf("q_chapter:%s", id)
	var chapter string
	if err := s.cache.Get(ctx, cacheKey, &chapter); err == nil {
		return chapter, nil
	}

	q, err := s.questionRepo.GetByIDWithoutEmbedding(ctx, id)
	if err != nil {
		return "", err
	}
	_ = s.cache.Set(ctx, cacheKey, q.Chapter, 24*time.Hour)
	return q.Chapter, nil
}

func (s *LearnService) loadQuestionChaptersBatch(ctx context.Context, ids []string) (map[string]string, error) {
	if len(ids) == 0 {
		return map[string]string{}, nil
	}

	result := make(map[string]string, len(ids))

	var uncached []string
	for _, id := range ids {
		cacheKey := fmt.Sprintf("q_chapter:%s", id)
		var chapter string
		if err := s.cache.Get(ctx, cacheKey, &chapter); err == nil {
			result[id] = chapter
		} else {
			uncached = append(uncached, id)
		}
	}

	if len(uncached) == 0 {
		return result, nil
	}

	rows, err := s.pool.Query(ctx, `SELECT id, chapter FROM questions WHERE id = ANY($1)`, uncached)
	if err != nil {
		return nil, fmt.Errorf("learn: batch load chapters: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, chapter string
		if err := rows.Scan(&id, &chapter); err != nil {
			return nil, fmt.Errorf("learn: scan chapter: %w", err)
		}
		result[id] = chapter
		_ = s.cache.Set(ctx, fmt.Sprintf("q_chapter:%s", id), chapter, 24*time.Hour)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("learn: rows error: %w", err)
	}

	return result, nil
}

func (s *LearnService) loadQuestionSubjectsBatch(ctx context.Context, ids []string) (map[string]string, error) {
	if len(ids) == 0 {
		return map[string]string{}, nil
	}

	result := make(map[string]string, len(ids))

	var uncached []string
	for _, id := range ids {
		cacheKey := fmt.Sprintf("q_subject:%s", id)
		var subject string
		if err := s.cache.Get(ctx, cacheKey, &subject); err == nil {
			result[id] = subject
		} else {
			uncached = append(uncached, id)
		}
	}

	if len(uncached) == 0 {
		return result, nil
	}

	rows, err := s.pool.Query(ctx, `SELECT id, subject FROM questions WHERE id = ANY($1)`, uncached)
	if err != nil {
		return nil, fmt.Errorf("learn: batch load subjects: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, subject string
		if err := rows.Scan(&id, &subject); err != nil {
			return nil, fmt.Errorf("learn: scan subject: %w", err)
		}
		result[id] = subject
		_ = s.cache.Set(ctx, fmt.Sprintf("q_subject:%s", id), subject, 24*time.Hour)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("learn: rows error: %w", err)
	}

	return result, nil
}

func (s *LearnService) loadCurrentQuestion(ctx context.Context, session *models.LearnSession) (*models.Question, error) {
	if session.CurrentQuestionID == nil {
		return nil, fmt.Errorf("session has no current question")
	}
	return s.loadQuestionByID(ctx, *session.CurrentQuestionID)
}

func (s *LearnService) EnrichAttemptCount(ctx context.Context, userID string, qp *models.QuestionPayload) {
	count, err := s.questionStats.GetAttemptCount(ctx, userID, qp.ID)
	if err != nil {
		slog.Debug("enrich attempt count", "question_id", qp.ID, "error", err)
		return
	}
	qp.AttemptCount = count
}
