package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"transverse/internal/cache"
	"transverse/internal/models"
)

// ProblemStatsRepo manages per-user per-problem historical attempts and solve records.
type ProblemStatsRepo struct {
	pool  *pgxpool.Pool
	cache cache.Cache
}

// NewProblemStatsRepo constructs a new ProblemStatsRepo.
func NewProblemStatsRepo(pool *pgxpool.Pool, c cache.Cache) *ProblemStatsRepo {
	return &ProblemStatsRepo{
		pool:  pool,
		cache: c,
	}
}

// Get retrieves the stats record for a specific user and problem.
func (r *ProblemStatsRepo) Get(ctx context.Context, userID, problemID string) (*models.UserProblemStats, error) {
	var s models.UserProblemStats
	err := r.pool.QueryRow(ctx, `
		SELECT user_id, problem_id, attempt_count, correct_count, total_time_ms, last_attempted
		FROM user_problem_stats
		WHERE user_id = $1 AND problem_id = $2
	`, userID, problemID).Scan(
		&s.UserID, &s.ProblemID, &s.AttemptCount, &s.CorrectCount, &s.TotalTimeMs, &s.LastAttempted,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("problem_stats_repo: get (%s, %s): %w", userID, problemID, err)
	}

	return &s, nil
}

// GetByUser retrieves all problem stat records for a user.
func (r *ProblemStatsRepo) GetByUser(ctx context.Context, userID string) ([]models.UserProblemStats, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT user_id, problem_id, attempt_count, correct_count, total_time_ms, last_attempted
		FROM user_problem_stats
		WHERE user_id = $1
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("problem_stats_repo: get by user %q: %w", userID, err)
	}
	defer rows.Close()

	var stats []models.UserProblemStats
	for rows.Next() {
		var s models.UserProblemStats
		if err := rows.Scan(
			&s.UserID, &s.ProblemID, &s.AttemptCount, &s.CorrectCount, &s.TotalTimeMs, &s.LastAttempted,
		); err != nil {
			return nil, fmt.Errorf("problem_stats_repo: scan problem stat: %w", err)
		}
		stats = append(stats, s)
	}

	return stats, rows.Err()
}

// GetAttemptCountsByUser returns a map of problemID -> attemptCount for a user, using cache if available.
func (r *ProblemStatsRepo) GetAttemptCountsByUser(ctx context.Context, userID string) (map[string]int, error) {
	cacheKey := fmt.Sprintf("seen:%s", userID)
	if r.cache != nil {
		var cached map[string]int
		if err := r.cache.Get(ctx, cacheKey, &cached); err == nil {
			return cached, nil
		}
	}

	rows, err := r.pool.Query(ctx, `
		SELECT problem_id, attempt_count
		FROM user_problem_stats
		WHERE user_id = $1
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("problem_stats_repo: get attempt counts: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var pid string
		var count int
		if err := rows.Scan(&pid, &count); err != nil {
			return nil, fmt.Errorf("problem_stats_repo: scan count: %w", err)
		}
		counts[pid] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("problem_stats_repo: iterate count rows: %w", err)
	}

	if r.cache != nil {
		_ = r.cache.Set(ctx, cacheKey, counts, 5*time.Minute)
	}

	return counts, nil
}

// RecordAttempt atomically increments the attempt count and updates timing / correctness for a user and problem.
func (r *ProblemStatsRepo) RecordAttempt(ctx context.Context, userID, problemID string, isCorrect bool, timeTakenMs int64) error {
	correctInc := 0
	if isCorrect {
		correctInc = 1
	}

	_, err := r.pool.Exec(ctx, `
		INSERT INTO user_problem_stats (
			user_id, problem_id, attempt_count, correct_count, total_time_ms, last_attempted
		) VALUES (
			$1, $2, 1, $3, $4, NOW()
		)
		ON CONFLICT (user_id, problem_id) DO UPDATE SET
			attempt_count = user_problem_stats.attempt_count + 1,
			correct_count = user_problem_stats.correct_count + $3,
			total_time_ms = user_problem_stats.total_time_ms + $4,
			last_attempted = NOW()
	`, userID, problemID, correctInc, timeTakenMs)
	if err != nil {
		return fmt.Errorf("problem_stats_repo: record attempt: %w", err)
	}

	if r.cache != nil {
		_ = r.cache.Del(ctx, fmt.Sprintf("seen:%s", userID))
	}

	return nil
}
