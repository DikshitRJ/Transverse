package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"transverse/internal/models"
)

type IngestRepo struct {
	pool *pgxpool.Pool
}

func NewIngestRepo(pool *pgxpool.Pool) *IngestRepo {
	return &IngestRepo{pool: pool}
}

func (r *IngestRepo) UpsertTutorial(ctx context.Context, t *models.TutorialIngestRecord, topicID string) (string, error) {
	query := `
		INSERT INTO tutorials (
			source, source_url, title, topic_id, topic_tags, type, difficulty, estimated_minutes, summary, license_note, thumbnail_url
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
		ON CONFLICT (source_url) DO UPDATE SET
			title = EXCLUDED.title,
			topic_id = EXCLUDED.topic_id,
			topic_tags = EXCLUDED.topic_tags,
			type = EXCLUDED.type,
			difficulty = EXCLUDED.difficulty,
			estimated_minutes = EXCLUDED.estimated_minutes,
			summary = EXCLUDED.summary,
			license_note = EXCLUDED.license_note,
			thumbnail_url = EXCLUDED.thumbnail_url,
			scraped_at = NOW()
		RETURNING id;
	`
	var id string
	var dbTopicID interface{}
	if topicID != "" {
		dbTopicID = topicID
	}

	err := r.pool.QueryRow(ctx, query,
		t.Source, t.SourceURL, t.Title, dbTopicID, t.TopicTags, t.Type, t.Difficulty,
		t.EstimatedMinutes, t.Summary, t.LicenseNote, t.ThumbnailURL,
	).Scan(&id)

	if err != nil {
		return "", fmt.Errorf("failed to upsert tutorial %s: %w", t.SourceURL, err)
	}

	return id, nil
}

func (r *IngestRepo) GetTutorialIDsByURLs(ctx context.Context, urls []string) (map[string]string, error) {
	if len(urls) == 0 {
		return nil, nil
	}
	
	query := `SELECT source_url, id FROM tutorials WHERE source_url = ANY($1)`
	rows, err := r.pool.Query(ctx, query, urls)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make(map[string]string)
	for rows.Next() {
		var url, id string
		if err := rows.Scan(&url, &id); err == nil {
			res[url] = id
		}
	}
	return res, nil
}

func (r *IngestRepo) CreateRoadmapTemplate(ctx context.Context, rt *models.RoadmapTemplateIngestRecord, resolveTopic func(string) string, getTutorials func([]string) []string) (string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var templateID string
	err = tx.QueryRow(ctx, `
		INSERT INTO roadmap_templates (target_role, source) 
		VALUES ($1, 'curated') RETURNING id`, 
		rt.TargetRole,
	).Scan(&templateID)
	if err != nil {
		return "", fmt.Errorf("failed to create roadmap template: %w", err)
	}

	for _, phase := range rt.Phases {
		var phaseID string
		unlockRuleBytes, _ := json.Marshal(phase.UnlockRule)
		
		err = tx.QueryRow(ctx, `
			INSERT INTO roadmap_phases (roadmap_template_id, sequence, title, unlock_rule) 
			VALUES ($1, $2, $3, $4) RETURNING id`,
			templateID, phase.Sequence, phase.Title, string(unlockRuleBytes),
		).Scan(&phaseID)
		if err != nil {
			return "", fmt.Errorf("failed to create roadmap phase %d: %w", phase.Sequence, err)
		}

		for _, node := range phase.Nodes {
			topicID := resolveTopic(node.TopicTag)
			if topicID == "" {
				topicID = node.TopicTag // fallback to raw string if cannot resolve, or fail? Let's use raw since it might be a valid ID already
			}

			tutorialIDs := getTutorials(node.TutorialSourceURLs)
			
			_, err = tx.Exec(ctx, `
				INSERT INTO roadmap_nodes (phase_id, topic_id, sequence, tutorial_ids, practice_topic_ids)
				VALUES ($1, $2, $3, $4, $5)`,
				phaseID, topicID, node.Sequence, tutorialIDs, node.PracticeTopicTags,
			)
			if err != nil {
				return "", fmt.Errorf("failed to create roadmap node %s: %w", node.TopicTag, err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return templateID, nil
}
