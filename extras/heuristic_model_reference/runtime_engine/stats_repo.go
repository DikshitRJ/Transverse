package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"velocity/internal/models"
)

type StatsRepo struct {
	pool *pgxpool.Pool
}

func NewStatsRepo(pool *pgxpool.Pool) *StatsRepo {
	return &StatsRepo{pool: pool}
}

func (r *StatsRepo) Get(ctx context.Context, userID string) (*models.LearningStats, error) {
	var ls models.LearningStats
	err := r.pool.QueryRow(ctx, `
		SELECT user_id, chapters, last_seen_at, updated_at
		FROM learning_stats WHERE user_id = $1
	`, userID).Scan(&ls.UserID, &ls.ChaptersRaw, &ls.LastSeenAt, &ls.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &models.LearningStats{UserID: userID}, nil
		}
		return nil, fmt.Errorf("stats_repo: get %q: %w", userID, err)
	}
	return &ls, nil
}

func (r *StatsRepo) GetChapterStats(ctx context.Context, userID, chapter string) (*models.ChapterStats, error) {
	ls, err := r.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	chapters, err := ls.Chapters()
	if err != nil {
		return nil, err
	}
	cs, ok := chapters[chapter]
	if !ok {
		return nil, fmt.Errorf("stats_repo: no stats for chapter %q", chapter)
	}
	return &cs, nil
}

func (r *StatsRepo) UpsertChapterStatsTx(ctx context.Context, tx pgx.Tx, userID, chapter string, cs models.ChapterStats) error {
	row := tx.QueryRow(ctx, `SELECT chapters FROM learning_stats WHERE user_id = $1 FOR UPDATE`, userID)

	var chaptersMap map[string]models.ChapterStats
	var raw json.RawMessage
	if err := row.Scan(&raw); err != nil {
		chaptersMap = map[string]models.ChapterStats{chapter: cs}
	} else if len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &chaptersMap); err != nil {
			chaptersMap = map[string]models.ChapterStats{}
		}
	} else {
		chaptersMap = map[string]models.ChapterStats{}
	}
	chaptersMap[chapter] = cs

	updated, err := json.Marshal(chaptersMap)
	if err != nil {
		return fmt.Errorf("stats_repo: marshal chapters: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO learning_stats (user_id, chapters, last_seen_at, updated_at)
		VALUES ($1, $2::jsonb, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT (user_id) DO UPDATE SET
			chapters = $2::jsonb,
			last_seen_at = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP
	`, userID, updated)
	if err != nil {
		return fmt.Errorf("stats_repo: upsert chapter stats: %w", err)
	}
	return nil
}

func (r *StatsRepo) GetChapterAvgTimeMs(ctx context.Context, userID, chapter string) (int64, error) {
	cs, err := r.GetChapterStats(ctx, userID, chapter)
	if err != nil {
		return 0, err
	}
	return cs.AvgTimeMs, nil
}
