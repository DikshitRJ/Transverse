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

// StatsRepo manages database operations for TopicStats entities.
type StatsRepo struct {
	pool  *pgxpool.Pool
	cache cache.Cache
}

// NewStatsRepo constructs a new StatsRepo.
func NewStatsRepo(pool *pgxpool.Pool, c cache.Cache) *StatsRepo {
	return &StatsRepo{
		pool:  pool,
		cache: c,
	}
}

// GetByUser retrieves all topic mastery stats for a user.
func (r *StatsRepo) GetByUser(ctx context.Context, userID string) ([]models.TopicStats, error) {
	cacheKey := fmt.Sprintf("topic_stats:%s", userID)
	if r.cache != nil {
		var cached []models.TopicStats
		if err := r.cache.Get(ctx, cacheKey, &cached); err == nil {
			return cached, nil
		}
	}

	rows, err := r.pool.Query(ctx, `
		SELECT user_id, topic, theta, mastery_score, glicko_rating, attempt_count, correct_count
		FROM topic_stats
		WHERE user_id = $1
		ORDER BY mastery_score DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("stats_repo: get by user %q: %w", userID, err)
	}
	defer rows.Close()

	var stats []models.TopicStats
	for rows.Next() {
		var s models.TopicStats
		if err := rows.Scan(
			&s.UserID, &s.Topic, &s.Theta, &s.MasteryScore, &s.GlickoRating,
			&s.AttemptCount, &s.CorrectCount,
		); err != nil {
			return nil, fmt.Errorf("stats_repo: scan topic stats: %w", err)
		}
		stats = append(stats, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("stats_repo: iterate rows: %w", err)
	}

	if r.cache != nil && len(stats) > 0 {
		_ = r.cache.Set(ctx, cacheKey, stats, 60*time.Second)
	}

	return stats, nil
}

// GetByUserAndTopic retrieves a single topic's stats for a user.
func (r *StatsRepo) GetByUserAndTopic(ctx context.Context, userID, topic string) (*models.TopicStats, error) {
	var s models.TopicStats
	err := r.pool.QueryRow(ctx, `
		SELECT user_id, topic, theta, mastery_score, glicko_rating, attempt_count, correct_count
		FROM topic_stats
		WHERE user_id = $1 AND topic = $2
	`, userID, topic).Scan(
		&s.UserID, &s.Topic, &s.Theta, &s.MasteryScore, &s.GlickoRating,
		&s.AttemptCount, &s.CorrectCount,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("stats_repo: get by user and topic (%s, %s): %w", userID, topic, err)
	}

	return &s, nil
}

// Upsert inserts or updates topic mastery statistics for a user.
func (r *StatsRepo) Upsert(ctx context.Context, s *models.TopicStats) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO topic_stats (
			user_id, topic, theta, mastery_score, glicko_rating, attempt_count, correct_count, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, NOW(), NOW()
		)
		ON CONFLICT (user_id, topic) DO UPDATE SET
			theta = EXCLUDED.theta,
			mastery_score = EXCLUDED.mastery_score,
			glicko_rating = EXCLUDED.glicko_rating,
			attempt_count = EXCLUDED.attempt_count,
			correct_count = EXCLUDED.correct_count,
			updated_at = NOW()
	`, s.UserID, s.Topic, s.Theta, s.MasteryScore, s.GlickoRating, s.AttemptCount, s.CorrectCount)
	if err != nil {
		return fmt.Errorf("stats_repo: upsert: %w", err)
	}

	if r.cache != nil {
		_ = r.cache.Del(ctx, fmt.Sprintf("topic_stats:%s", s.UserID))
	}

	return nil
}

// BatchUpsert saves multiple topic statistics inside a single transaction.
func (r *StatsRepo) BatchUpsert(ctx context.Context, stats []models.TopicStats) error {
	if len(stats) == 0 {
		return nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("stats_repo: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, s := range stats {
		_, err := tx.Exec(ctx, `
			INSERT INTO topic_stats (
				user_id, topic, theta, mastery_score, glicko_rating, attempt_count, correct_count, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, NOW(), NOW()
			)
			ON CONFLICT (user_id, topic) DO UPDATE SET
				theta = EXCLUDED.theta,
				mastery_score = EXCLUDED.mastery_score,
				glicko_rating = EXCLUDED.glicko_rating,
				attempt_count = EXCLUDED.attempt_count,
				correct_count = EXCLUDED.correct_count,
				updated_at = NOW()
		`, s.UserID, s.Topic, s.Theta, s.MasteryScore, s.GlickoRating, s.AttemptCount, s.CorrectCount)
		if err != nil {
			return fmt.Errorf("stats_repo: batch upsert topic %q: %w", s.Topic, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("stats_repo: commit batch: %w", err)
	}

	if r.cache != nil && len(stats) > 0 {
		_ = r.cache.Del(ctx, fmt.Sprintf("topic_stats:%s", stats[0].UserID))
	}

	return nil
}
