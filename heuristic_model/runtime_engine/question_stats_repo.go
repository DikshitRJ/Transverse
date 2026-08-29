package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type QuestionStatsRepo struct {
	pool *pgxpool.Pool
}

func NewQuestionStatsRepo(pool *pgxpool.Pool) *QuestionStatsRepo {
	return &QuestionStatsRepo{pool: pool}
}

func (r *QuestionStatsRepo) UpsertTx(ctx context.Context, tx pgx.Tx, userID, questionID string, correct bool, timeTakenMs int64) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO user_question_stats (user_id, question_id, attempt_count, correct_count, total_time_ms, last_correct, last_seen_at)
		VALUES ($1, $2, 1, CASE WHEN $3 THEN 1 ELSE 0 END, $4, $3, CURRENT_TIMESTAMP)
		ON CONFLICT (user_id, question_id) DO UPDATE SET
			attempt_count = user_question_stats.attempt_count + 1,
			correct_count = user_question_stats.correct_count + CASE WHEN $3 THEN 1 ELSE 0 END,
			total_time_ms = user_question_stats.total_time_ms + $4,
			last_correct = $3,
			last_seen_at = CURRENT_TIMESTAMP
	`, userID, questionID, correct, timeTakenMs)
	if err != nil {
		return fmt.Errorf("question_stats_repo: upsert: %w", err)
	}
	return nil
}

func (r *QuestionStatsRepo) GetAllAttemptCounts(ctx context.Context, userID string) (map[string]int, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT question_id, attempt_count FROM user_question_stats WHERE user_id = $1
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("question_stats_repo: get all attempt counts: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var qid string
		var count int
		if err := rows.Scan(&qid, &count); err != nil {
			return nil, fmt.Errorf("question_stats_repo: scan: %w", err)
		}
		counts[qid] = count
	}
	return counts, rows.Err()
}

func (r *QuestionStatsRepo) GetAttemptCount(ctx context.Context, userID, questionID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(attempt_count, 0) FROM user_question_stats WHERE user_id = $1 AND question_id = $2
	`, userID, questionID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("question_stats_repo: get attempt count: %w", err)
	}
	return count, nil
}
