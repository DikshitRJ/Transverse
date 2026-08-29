// Package main provides database seeding and upsert capabilities for normalized problem vectors.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

const upsertProblemSQL = `
INSERT INTO problems (
    id, source, name, url, slug, contest_id, tags, topic, subtopic,
    difficulty_label, glicko_rating, glicko_rd, glicko_volatility, embedding, embed_text
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
)
ON CONFLICT (id) DO UPDATE SET
    source            = EXCLUDED.source,
    name              = EXCLUDED.name,
    url               = EXCLUDED.url,
    slug              = EXCLUDED.slug,
    contest_id        = EXCLUDED.contest_id,
    tags              = EXCLUDED.tags,
    topic             = EXCLUDED.topic,
    subtopic          = EXCLUDED.subtopic,
    difficulty_label  = EXCLUDED.difficulty_label,
    glicko_rating     = EXCLUDED.glicko_rating,
    glicko_rd         = EXCLUDED.glicko_rd,
    glicko_volatility = EXCLUDED.glicko_volatility,
    embedding         = COALESCE(EXCLUDED.embedding, problems.embedding),
    embed_text        = EXCLUDED.embed_text,
    updated_at        = NOW();
`

// SeedProblems upserts all normalized problems with their corresponding vector embeddings into PostgreSQL.
// It executes in chunked transactional batches of size batchSize.
func SeedProblems(ctx context.Context, pool *pgxpool.Pool, problems []NormalizedProblem, embeddings map[string][]float32, batchSize int) error {
	if pool == nil {
		return fmt.Errorf("database connection pool cannot be nil")
	}

	totalProblems := len(problems)
	if totalProblems == 0 {
		log.Printf("[seeder] no problems to seed")
		return nil
	}

	if batchSize <= 0 {
		batchSize = 250
	}

	log.Printf("[seeder] starting database upsert for %d problems (batch_size=%d)", totalProblems, batchSize)
	startTime := time.Now()
	seededCount := 0

	for i := 0; i < totalProblems; i += batchSize {
		end := i + batchSize
		if end > totalProblems {
			end = totalProblems
		}
		chunk := problems[i:end]

		if err := ctx.Err(); err != nil {
			return fmt.Errorf("seeding context cancelled: %w", err)
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("failed to begin transaction at offset %d: %w", i, err)
		}

		batch := &pgx.Batch{}
		for _, p := range chunk {
			var embValue interface{} = nil
			if emb, ok := embeddings[p.ID]; ok && len(emb) > 0 {
				embValue = pgvector.NewVector(emb)
			}

			batch.Queue(
				upsertProblemSQL,
				p.ID,
				p.Source,
				p.Name,
				p.URL,
				p.Slug,
				p.ContestID,
				p.Tags,
				p.Topic,
				p.Subtopic,
				p.DifficultyLabel,
				p.GlickoRating,
				p.GlickoRD,
				p.GlickoVolatility,
				embValue,
				p.EmbedText,
			)
		}

		br := tx.SendBatch(ctx, batch)
		batchSuccess := true
		var execErr error

		for range chunk {
			if _, err := br.Exec(); err != nil {
				batchSuccess = false
				execErr = err
				break
			}
		}

		closeErr := br.Close()
		if !batchSuccess || closeErr != nil {
			_ = tx.Rollback(ctx)
			if execErr != nil {
				return fmt.Errorf("failed executing batch upsert at range [%d:%d]: %w", i, end, execErr)
			}
			return fmt.Errorf("failed closing batch result at range [%d:%d]: %w", i, end, closeErr)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("failed to commit transaction at range [%d:%d]: %w", i, end, err)
		}

		seededCount += len(chunk)
		if seededCount%1000 == 0 || seededCount == totalProblems {
			pct := (float64(seededCount) / float64(totalProblems)) * 100.0
			log.Printf("[seeder] progress: %d/%d problems seeded (%.1f%%)", seededCount, totalProblems, pct)
		}
	}

	totalDuration := time.Since(startTime)
	log.Printf("[seeder] completed seeding %d problems to database in %s", seededCount, totalDuration.Round(time.Millisecond))
	return nil
}
