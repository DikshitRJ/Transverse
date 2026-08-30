package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"transverse/internal/models"
)

// EvidenceRepo persists evidence sources and their extracted signal to Postgres. It
// satisfies the evidence.EvidenceRepository interface (see internal/evidence/service.go).
type EvidenceRepo struct {
	pool *pgxpool.Pool
}

// NewEvidenceRepo constructs a new EvidenceRepo instance.
func NewEvidenceRepo(pool *pgxpool.Pool) *EvidenceRepo {
	return &EvidenceRepo{pool: pool}
}

// CreateSource inserts a new evidence source row.
func (r *EvidenceRepo) CreateSource(ctx context.Context, source *models.EvidenceSource) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO evidence_sources (id, user_id, kind, external_ref, object_key, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, source.ID, source.UserID, source.Kind, source.ExternalRef, source.ObjectKey, source.Status, source.CreatedAt)
	if err != nil {
		return fmt.Errorf("evidence_repo: create source: %w", err)
	}
	return nil
}

// UpdateSourceStatus updates an evidence source's processing status and optional error
// message, stamping processed_at once the source reaches a terminal state.
func (r *EvidenceRepo) UpdateSourceStatus(ctx context.Context, id string, status models.EvidenceStatus, errMsg *string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE evidence_sources
		SET status = $2,
		    error_message = $3,
		    processed_at = CASE WHEN $2 IN ('done', 'failed') THEN NOW() ELSE processed_at END
		WHERE id = $1
	`, id, status, errMsg)
	if err != nil {
		return fmt.Errorf("evidence_repo: update source status: %w", err)
	}
	return nil
}

// ClearObjectKey nulls out the object_key once the underlying MinIO object has been
// deleted, so no dangling reference to a purged upload remains.
func (r *EvidenceRepo) ClearObjectKey(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `UPDATE evidence_sources SET object_key = NULL WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("evidence_repo: clear object key: %w", err)
	}
	return nil
}

// CreateExtract inserts the normalized signal extracted from an evidence source.
func (r *EvidenceRepo) CreateExtract(ctx context.Context, extract *models.EvidenceExtract) error {
	extractedJSON, err := json.Marshal(extract.ExtractedJSON)
	if err != nil {
		return fmt.Errorf("evidence_repo: marshal extracted json: %w", err)
	}

	_, err = r.pool.Exec(ctx, `
		INSERT INTO evidence_extracts (id, evidence_source_id, extracted_json, confidence, created_at)
		VALUES ($1, $2, $3::jsonb, $4, $5)
	`, extract.ID, extract.EvidenceSourceID, extractedJSON, extract.Confidence, extract.CreatedAt)
	if err != nil {
		return fmt.Errorf("evidence_repo: create extract: %w", err)
	}
	return nil
}

// GetSource fetches an evidence source by ID.
func (r *EvidenceRepo) GetSource(ctx context.Context, id string) (*models.EvidenceSource, error) {
	var s models.EvidenceSource
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, kind, external_ref, object_key, status, error_message, created_at, processed_at, purge_at
		FROM evidence_sources WHERE id = $1
	`, id).Scan(
		&s.ID, &s.UserID, &s.Kind, &s.ExternalRef, &s.ObjectKey, &s.Status, &s.ErrorMessage,
		&s.CreatedAt, &s.ProcessedAt, &s.PurgeAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("evidence source not found: %s", id)
		}
		return nil, fmt.Errorf("evidence_repo: get source: %w", err)
	}
	return &s, nil
}
