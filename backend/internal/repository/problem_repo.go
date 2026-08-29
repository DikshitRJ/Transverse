// Package repository provides database persistence methods for domain entities using pgxpool and pgvector.
package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"

	"transverse/internal/cache"
	"transverse/internal/models"
)

// ProblemRepo manages database access and caching for Problem entities.
type ProblemRepo struct {
	pool  *pgxpool.Pool
	cache cache.Cache
}

// NewProblemRepo constructs a new ProblemRepo instance.
func NewProblemRepo(pool *pgxpool.Pool, c cache.Cache) *ProblemRepo {
	return &ProblemRepo{
		pool:  pool,
		cache: c,
	}
}

// GetByID retrieves a single problem by ID, checking cache first.
func (r *ProblemRepo) GetByID(ctx context.Context, id string) (*models.Problem, error) {
	cacheKey := fmt.Sprintf("problem:%s", id)
	if r.cache != nil {
		var cached models.Problem
		if err := r.cache.Get(ctx, cacheKey, &cached); err == nil {
			return &cached, nil
		}
	}

	var p models.Problem
	var emb pgvector.Vector
	err := r.pool.QueryRow(ctx, `
		SELECT id, source, name, url, slug, contest_id, tags, topic, subtopic,
		       difficulty_label, glicko_rating, glicko_rd, glicko_volatility,
		       attempt_count, solve_rate, avg_time_ms, embedding, embed_text,
		       created_at, updated_at
		FROM problems WHERE id = $1
	`, id).Scan(
		&p.ID, &p.Source, &p.Name, &p.URL, &p.Slug, &p.ContestID, &p.Tags, &p.Topic, &p.Subtopic,
		&p.DifficultyLabel, &p.GlickoRating, &p.GlickoRD, &p.GlickoVolatility,
		&p.AttemptCount, &p.SolveRate, &p.AvgTimeMs, &emb, &p.EmbedText,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("problem not found: %s", id)
		}
		return nil, fmt.Errorf("problem_repo: get by id %q: %w", id, err)
	}

	p.Embedding = emb

	if r.cache != nil {
		_ = r.cache.Set(ctx, cacheKey, p, 24*time.Hour)
	}

	return &p, nil
}

// GetByTopic retrieves all problems matching a canonical topic name.
func (r *ProblemRepo) GetByTopic(ctx context.Context, topic string) ([]models.Problem, error) {
	cacheKey := fmt.Sprintf("problems:topic:%s", topic)
	if r.cache != nil {
		var cached []models.Problem
		if err := r.cache.Get(ctx, cacheKey, &cached); err == nil {
			return cached, nil
		}
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, source, name, url, slug, contest_id, tags, topic, subtopic,
		       difficulty_label, glicko_rating, glicko_rd, glicko_volatility,
		       attempt_count, solve_rate, avg_time_ms, embedding, embed_text,
		       created_at, updated_at
		FROM problems WHERE topic = $1
		ORDER BY glicko_rating ASC
	`, topic)
	if err != nil {
		return nil, fmt.Errorf("problem_repo: get by topic %q: %w", topic, err)
	}
	defer rows.Close()

	var problems []models.Problem
	for rows.Next() {
		var p models.Problem
		var emb pgvector.Vector
		if err := rows.Scan(
			&p.ID, &p.Source, &p.Name, &p.URL, &p.Slug, &p.ContestID, &p.Tags, &p.Topic, &p.Subtopic,
			&p.DifficultyLabel, &p.GlickoRating, &p.GlickoRD, &p.GlickoVolatility,
			&p.AttemptCount, &p.SolveRate, &p.AvgTimeMs, &emb, &p.EmbedText,
			&p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("problem_repo: scan problem: %w", err)
		}
		p.Embedding = emb
		problems = append(problems, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("problem_repo: iterate rows: %w", err)
	}

	if r.cache != nil && len(problems) > 0 {
		_ = r.cache.Set(ctx, cacheKey, problems, 10*time.Minute)
	}

	return problems, nil
}

// GetByScope retrieves problems within a specified SessionScope filter.
func (r *ProblemRepo) GetByScope(ctx context.Context, scope models.SessionScope) ([]models.Problem, error) {
	query := `
		SELECT id, source, name, url, slug, contest_id, tags, topic, subtopic,
		       difficulty_label, glicko_rating, glicko_rd, glicko_volatility,
		       attempt_count, solve_rate, avg_time_ms, embedding, embed_text,
		       created_at, updated_at
		FROM problems
		WHERE 1=1
	`
	args := make([]interface{}, 0)
	argIdx := 1

	if len(scope.Topics) > 0 {
		query += fmt.Sprintf(" AND topic = ANY($%d)", argIdx)
		args = append(args, scope.Topics)
		argIdx++
	}

	if len(scope.Subtopics) > 0 {
		query += fmt.Sprintf(" AND subtopic = ANY($%d)", argIdx)
		args = append(args, scope.Subtopics)
		argIdx++
	}

	if len(scope.Sources) > 0 {
		query += fmt.Sprintf(" AND source = ANY($%d)", argIdx)
		args = append(args, scope.Sources)
		argIdx++
	}

	if scope.DifficultyRange[0] > 0 || scope.DifficultyRange[1] > 0 {
		minDiff := scope.DifficultyRange[0]
		maxDiff := scope.DifficultyRange[1]
		if minDiff > 0 {
			query += fmt.Sprintf(" AND glicko_rating >= $%d", argIdx)
			args = append(args, float64(minDiff))
			argIdx++
		}
		if maxDiff > 0 {
			query += fmt.Sprintf(" AND glicko_rating <= $%d", argIdx)
			args = append(args, float64(maxDiff))
			argIdx++
		}
	}

	query += " ORDER BY glicko_rating ASC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("problem_repo: get by scope: %w", err)
	}
	defer rows.Close()

	var problems []models.Problem
	for rows.Next() {
		var p models.Problem
		var emb pgvector.Vector
		if err := rows.Scan(
			&p.ID, &p.Source, &p.Name, &p.URL, &p.Slug, &p.ContestID, &p.Tags, &p.Topic, &p.Subtopic,
			&p.DifficultyLabel, &p.GlickoRating, &p.GlickoRD, &p.GlickoVolatility,
			&p.AttemptCount, &p.SolveRate, &p.AvgTimeMs, &emb, &p.EmbedText,
			&p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("problem_repo: scan scope problem: %w", err)
		}
		p.Embedding = emb
		problems = append(problems, p)
	}

	return problems, rows.Err()
}

// FindSimilar performs an approximate nearest-neighbor search using vector cosine similarity.
func (r *ProblemRepo) FindSimilar(ctx context.Context, embedding pgvector.Vector, topic string, limit int) ([]models.Problem, error) {
	if limit <= 0 {
		limit = 5
	}

	query := `
		SELECT id, source, name, url, slug, contest_id, tags, topic, subtopic,
		       difficulty_label, glicko_rating, glicko_rd, glicko_volatility,
		       attempt_count, solve_rate, avg_time_ms, embedding, embed_text,
		       created_at, updated_at
		FROM problems
	`
	args := []interface{}{embedding}
	if topic != "" {
		query += " WHERE topic = $2 ORDER BY embedding <=> $1 LIMIT $3"
		args = append(args, topic, limit)
	} else {
		query += " ORDER BY embedding <=> $1 LIMIT $2"
		args = append(args, limit)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("problem_repo: find similar: %w", err)
	}
	defer rows.Close()

	var problems []models.Problem
	for rows.Next() {
		var p models.Problem
		var emb pgvector.Vector
		if err := rows.Scan(
			&p.ID, &p.Source, &p.Name, &p.URL, &p.Slug, &p.ContestID, &p.Tags, &p.Topic, &p.Subtopic,
			&p.DifficultyLabel, &p.GlickoRating, &p.GlickoRD, &p.GlickoVolatility,
			&p.AttemptCount, &p.SolveRate, &p.AvgTimeMs, &emb, &p.EmbedText,
			&p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("problem_repo: scan similar problem: %w", err)
		}
		p.Embedding = emb
		problems = append(problems, p)
	}

	return problems, rows.Err()
}

// Search executes filtered search across problem names, tags, sources, topics, and difficulties.
func (r *ProblemRepo) Search(ctx context.Context, req models.ProblemSearchRequest) ([]models.Problem, int, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := req.Offset
	if offset < 0 {
		offset = 0
	}

	whereClauses := []string{"1=1"}
	args := make([]interface{}, 0)
	argIdx := 1

	if strings.TrimSpace(req.Query) != "" {
		q := "%" + strings.ToLower(strings.TrimSpace(req.Query)) + "%"
		whereClauses = append(whereClauses, fmt.Sprintf("(LOWER(name) LIKE $%d OR LOWER(slug) LIKE $%d OR $%d = ANY(tags))", argIdx, argIdx, strings.ToLower(strings.TrimSpace(req.Query))))
		args = append(args, q)
		argIdx++
	}

	if strings.TrimSpace(req.Topic) != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("topic = $%d", argIdx))
		args = append(args, strings.TrimSpace(req.Topic))
		argIdx++
	}

	if strings.TrimSpace(req.Source) != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("source = $%d", argIdx))
		args = append(args, strings.TrimSpace(req.Source))
		argIdx++
	}

	if strings.TrimSpace(req.DifficultyLabel) != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("difficulty_label = $%d", argIdx))
		args = append(args, strings.TrimSpace(req.DifficultyLabel))
		argIdx++
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM problems WHERE %s", whereSQL)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("problem_repo: count search: %w", err)
	}

	dataQuery := fmt.Sprintf(`
		SELECT id, source, name, url, slug, contest_id, tags, topic, subtopic,
		       difficulty_label, glicko_rating, glicko_rd, glicko_volatility,
		       attempt_count, solve_rate, avg_time_ms, embedding, embed_text,
		       created_at, updated_at
		FROM problems
		WHERE %s
		ORDER BY glicko_rating ASC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argIdx, argIdx+1)

	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("problem_repo: query search: %w", err)
	}
	defer rows.Close()

	var problems []models.Problem
	for rows.Next() {
		var p models.Problem
		var emb pgvector.Vector
		if err := rows.Scan(
			&p.ID, &p.Source, &p.Name, &p.URL, &p.Slug, &p.ContestID, &p.Tags, &p.Topic, &p.Subtopic,
			&p.DifficultyLabel, &p.GlickoRating, &p.GlickoRD, &p.GlickoVolatility,
			&p.AttemptCount, &p.SolveRate, &p.AvgTimeMs, &emb, &p.EmbedText,
			&p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("problem_repo: scan search row: %w", err)
		}
		p.Embedding = emb
		problems = append(problems, p)
	}

	return problems, total, rows.Err()
}

// Upsert inserts or updates a single problem.
func (r *ProblemRepo) Upsert(ctx context.Context, p *models.Problem) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO problems (
			id, source, name, url, slug, contest_id, tags, topic, subtopic,
			difficulty_label, glicko_rating, glicko_rd, glicko_volatility,
			attempt_count, solve_rate, avg_time_ms, embedding, embed_text,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, NOW(), NOW()
		)
		ON CONFLICT (id) DO UPDATE SET
			source = EXCLUDED.source,
			name = EXCLUDED.name,
			url = EXCLUDED.url,
			slug = EXCLUDED.slug,
			contest_id = EXCLUDED.contest_id,
			tags = EXCLUDED.tags,
			topic = EXCLUDED.topic,
			subtopic = EXCLUDED.subtopic,
			difficulty_label = EXCLUDED.difficulty_label,
			glicko_rating = EXCLUDED.glicko_rating,
			glicko_rd = EXCLUDED.glicko_rd,
			glicko_volatility = EXCLUDED.glicko_volatility,
			attempt_count = EXCLUDED.attempt_count,
			solve_rate = EXCLUDED.solve_rate,
			avg_time_ms = EXCLUDED.avg_time_ms,
			embedding = EXCLUDED.embedding,
			embed_text = EXCLUDED.embed_text,
			updated_at = NOW()
	`,
		p.ID, p.Source, p.Name, p.URL, p.Slug, p.ContestID, p.Tags, p.Topic, p.Subtopic,
		p.DifficultyLabel, p.GlickoRating, p.GlickoRD, p.GlickoVolatility,
		p.AttemptCount, p.SolveRate, p.AvgTimeMs, p.Embedding, p.EmbedText,
	)
	if err != nil {
		return fmt.Errorf("problem_repo: upsert: %w", err)
	}

	if r.cache != nil {
		_ = r.cache.Del(ctx, fmt.Sprintf("problem:%s", p.ID))
		_ = r.cache.Del(ctx, fmt.Sprintf("problems:topic:%s", p.Topic))
	}

	return nil
}

// BatchUpsert performs a batch insert or update across multiple problems inside a single transaction.
func (r *ProblemRepo) BatchUpsert(ctx context.Context, problems []models.Problem) error {
	if len(problems) == 0 {
		return nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("problem_repo: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, p := range problems {
		_, err := tx.Exec(ctx, `
			INSERT INTO problems (
				id, source, name, url, slug, contest_id, tags, topic, subtopic,
				difficulty_label, glicko_rating, glicko_rd, glicko_volatility,
				attempt_count, solve_rate, avg_time_ms, embedding, embed_text,
				created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, NOW(), NOW()
			)
			ON CONFLICT (id) DO UPDATE SET
				source = EXCLUDED.source,
				name = EXCLUDED.name,
				url = EXCLUDED.url,
				slug = EXCLUDED.slug,
				contest_id = EXCLUDED.contest_id,
				tags = EXCLUDED.tags,
				topic = EXCLUDED.topic,
				subtopic = EXCLUDED.subtopic,
				difficulty_label = EXCLUDED.difficulty_label,
				glicko_rating = EXCLUDED.glicko_rating,
				glicko_rd = EXCLUDED.glicko_rd,
				glicko_volatility = EXCLUDED.glicko_volatility,
				attempt_count = EXCLUDED.attempt_count,
				solve_rate = EXCLUDED.solve_rate,
				avg_time_ms = EXCLUDED.avg_time_ms,
				embedding = EXCLUDED.embedding,
				embed_text = EXCLUDED.embed_text,
				updated_at = NOW()
		`,
			p.ID, p.Source, p.Name, p.URL, p.Slug, p.ContestID, p.Tags, p.Topic, p.Subtopic,
			p.DifficultyLabel, p.GlickoRating, p.GlickoRD, p.GlickoVolatility,
			p.AttemptCount, p.SolveRate, p.AvgTimeMs, p.Embedding, p.EmbedText,
		)
		if err != nil {
			return fmt.Errorf("problem_repo: batch upsert item %q: %w", p.ID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("problem_repo: commit batch: %w", err)
	}

	return nil
}

// IncrementAttemptCount updates global problem counters after an attempt.
func (r *ProblemRepo) IncrementAttemptCount(ctx context.Context, problemID string, isCorrect bool) error {
	correctInc := 0
	if isCorrect {
		correctInc = 1
	}

	_, err := r.pool.Exec(ctx, `
		UPDATE problems
		SET attempt_count = attempt_count + 1,
		    solve_rate = CASE
		        WHEN attempt_count + 1 > 0 THEN ((solve_rate * attempt_count) + $2) / (attempt_count + 1)
		        ELSE solve_rate
		    END,
		    updated_at = NOW()
		WHERE id = $1
	`, problemID, correctInc)
	if err != nil {
		return fmt.Errorf("problem_repo: increment attempt count: %w", err)
	}

	if r.cache != nil {
		_ = r.cache.Del(ctx, fmt.Sprintf("problem:%s", problemID))
	}

	return nil
}
