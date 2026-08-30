package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"transverse/internal/models"
)

type RoadmapRepo struct {
	pool *pgxpool.Pool
}

func NewRoadmapRepo(pool *pgxpool.Pool) *RoadmapRepo {
	return &RoadmapRepo{pool: pool}
}

func (r *RoadmapRepo) CreateTemplate(ctx context.Context, tmpl *models.RoadmapTemplate) error {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO roadmap_templates (target_role, source, version)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`, tmpl.TargetRole, tmpl.Source, tmpl.Version).Scan(&tmpl.ID, &tmpl.CreatedAt)
	if err != nil {
		return fmt.Errorf("roadmap_repo: create template: %w", err)
	}
	return nil
}

func (r *RoadmapRepo) CreatePhase(ctx context.Context, phase *models.RoadmapPhase) error {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO roadmap_phases (roadmap_template_id, sequence, title, unlock_rule)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`, phase.RoadmapTemplateID, phase.Sequence, phase.Title, phase.UnlockRule).Scan(&phase.ID, &phase.CreatedAt)
	if err != nil {
		return fmt.Errorf("roadmap_repo: create phase: %w", err)
	}
	return nil
}

func (r *RoadmapRepo) CreateNode(ctx context.Context, node *models.RoadmapNode) error {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO roadmap_nodes (phase_id, topic_id, sequence, unlock_rule, tutorial_ids, practice_topic_ids)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`, node.PhaseID, node.TopicID, node.Sequence, node.UnlockRule, node.TutorialIDs, node.PracticeTopicIDs).Scan(&node.ID, &node.CreatedAt)
	if err != nil {
		return fmt.Errorf("roadmap_repo: create node: %w", err)
	}
	return nil
}

func (r *RoadmapRepo) CreateUserRoadmap(ctx context.Context, ur *models.UserRoadmap) error {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO user_roadmaps (user_id, roadmap_template_id, status, current_phase_id)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id) DO UPDATE SET roadmap_template_id = EXCLUDED.roadmap_template_id, status = EXCLUDED.status, current_phase_id = EXCLUDED.current_phase_id
		RETURNING id, created_at
	`, ur.UserID, ur.RoadmapTemplateID, ur.Status, ur.CurrentPhaseID).Scan(&ur.ID, &ur.CreatedAt)
	if err != nil {
		return fmt.Errorf("roadmap_repo: create user roadmap: %w", err)
	}
	return nil
}

func (r *RoadmapRepo) CreateUserNodeProgress(ctx context.Context, up *models.UserRoadmapNodeProgress) error {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO user_roadmap_node_progress (user_roadmap_id, node_id, status)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_roadmap_id, node_id) DO UPDATE SET status = EXCLUDED.status
		RETURNING id, unlocked_at, mastered_at
	`, up.UserRoadmapID, up.NodeID, up.Status).Scan(&up.ID, &up.UnlockedAt, &up.MasteredAt)
	if err != nil {
		return fmt.Errorf("roadmap_repo: create user node progress: %w", err)
	}
	return nil
}

func (r *RoadmapRepo) GetUserRoadmap(ctx context.Context, userID uuid.UUID) (*models.UserRoadmap, error) {
	var ur models.UserRoadmap
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, roadmap_template_id, status, current_phase_id, created_at
		FROM user_roadmaps WHERE user_id = $1
	`, userID).Scan(&ur.ID, &ur.UserID, &ur.RoadmapTemplateID, &ur.Status, &ur.CurrentPhaseID, &ur.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // not found
		}
		return nil, fmt.Errorf("roadmap_repo: get user roadmap: %w", err)
	}
	return &ur, nil
}

func (r *RoadmapRepo) GetTemplatePhases(ctx context.Context, templateID uuid.UUID) ([]models.RoadmapPhase, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, roadmap_template_id, sequence, title, unlock_rule, created_at
		FROM roadmap_phases WHERE roadmap_template_id = $1 ORDER BY sequence ASC
	`, templateID)
	if err != nil {
		return nil, fmt.Errorf("roadmap_repo: get template phases: %w", err)
	}
	defer rows.Close()

	var phases []models.RoadmapPhase
	for rows.Next() {
		var p models.RoadmapPhase
		if err := rows.Scan(&p.ID, &p.RoadmapTemplateID, &p.Sequence, &p.Title, &p.UnlockRule, &p.CreatedAt); err != nil {
			return nil, err
		}
		phases = append(phases, p)
	}
	return phases, nil
}

func (r *RoadmapRepo) GetPhaseNodes(ctx context.Context, phaseID uuid.UUID) ([]models.RoadmapNode, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, phase_id, topic_id, sequence, unlock_rule, tutorial_ids, practice_topic_ids, created_at
		FROM roadmap_nodes WHERE phase_id = $1 ORDER BY sequence ASC
	`, phaseID)
	if err != nil {
		return nil, fmt.Errorf("roadmap_repo: get phase nodes: %w", err)
	}
	defer rows.Close()

	var nodes []models.RoadmapNode
	for rows.Next() {
		var n models.RoadmapNode
		if err := rows.Scan(&n.ID, &n.PhaseID, &n.TopicID, &n.Sequence, &n.UnlockRule, &n.TutorialIDs, &n.PracticeTopicIDs, &n.CreatedAt); err != nil {
			return nil, err
		}
		nodes = append(nodes, n)
	}
	return nodes, nil
}

func (r *RoadmapRepo) GetUserNodeProgresses(ctx context.Context, userRoadmapID uuid.UUID) ([]models.UserRoadmapNodeProgress, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_roadmap_id, node_id, status, unlocked_at, mastered_at
		FROM user_roadmap_node_progress WHERE user_roadmap_id = $1
	`, userRoadmapID)
	if err != nil {
		return nil, fmt.Errorf("roadmap_repo: get user node progresses: %w", err)
	}
	defer rows.Close()

	var progresses []models.UserRoadmapNodeProgress
	for rows.Next() {
		var p models.UserRoadmapNodeProgress
		if err := rows.Scan(&p.ID, &p.UserRoadmapID, &p.NodeID, &p.Status, &p.UnlockedAt, &p.MasteredAt); err != nil {
			return nil, err
		}
		progresses = append(progresses, p)
	}
	return progresses, nil
}

func (r *RoadmapRepo) UpdateNodeProgress(ctx context.Context, id uuid.UUID, status models.NodeStatus, unlockedAt, masteredAt *time.Time) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE user_roadmap_node_progress
		SET status = $2, unlocked_at = $3, mastered_at = $4
		WHERE id = $1
	`, id, status, unlockedAt, masteredAt)
	if err != nil {
		return fmt.Errorf("roadmap_repo: update node progress: %w", err)
	}
	return nil
}

func (r *RoadmapRepo) UpdateUserRoadmapPhase(ctx context.Context, id uuid.UUID, phaseID *uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE user_roadmaps
		SET current_phase_id = $2
		WHERE id = $1
	`, id, phaseID)
	if err != nil {
		return fmt.Errorf("roadmap_repo: update user roadmap phase: %w", err)
	}
	return nil
}

func (r *RoadmapRepo) GetPhaseByID(ctx context.Context, phaseID uuid.UUID) (*models.RoadmapPhase, error) {
	var p models.RoadmapPhase
	err := r.pool.QueryRow(ctx, `
		SELECT id, roadmap_template_id, sequence, title, unlock_rule, created_at
		FROM roadmap_phases WHERE id = $1
	`, phaseID).Scan(&p.ID, &p.RoadmapTemplateID, &p.Sequence, &p.Title, &p.UnlockRule, &p.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("roadmap_repo: get phase by id: %w", err)
	}
	return &p, nil
}

func (r *RoadmapRepo) GetNodeByID(ctx context.Context, nodeID uuid.UUID) (*models.RoadmapNode, error) {
	var n models.RoadmapNode
	err := r.pool.QueryRow(ctx, `
		SELECT id, phase_id, topic_id, sequence, unlock_rule, tutorial_ids, practice_topic_ids, created_at
		FROM roadmap_nodes WHERE id = $1
	`, nodeID).Scan(&n.ID, &n.PhaseID, &n.TopicID, &n.Sequence, &n.UnlockRule, &n.TutorialIDs, &n.PracticeTopicIDs, &n.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("roadmap_repo: get node by id: %w", err)
	}
	return &n, nil
}

func (r *RoadmapRepo) GetUserProgressByNode(ctx context.Context, userRoadmapID, nodeID uuid.UUID) (*models.UserRoadmapNodeProgress, error) {
	var p models.UserRoadmapNodeProgress
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_roadmap_id, node_id, status, unlocked_at, mastered_at
		FROM user_roadmap_node_progress WHERE user_roadmap_id = $1 AND node_id = $2
	`, userRoadmapID, nodeID).Scan(&p.ID, &p.UserRoadmapID, &p.NodeID, &p.Status, &p.UnlockedAt, &p.MasteredAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("roadmap_repo: get user progress by node: %w", err)
	}
	return &p, nil
}

func (r *RoadmapRepo) GetCuratedTemplate(ctx context.Context) (*models.RoadmapTemplate, error) {
	var tmpl models.RoadmapTemplate
	err := r.pool.QueryRow(ctx, `
		SELECT id, target_role, source, version, created_at
		FROM roadmap_templates
		ORDER BY created_at DESC LIMIT 1
	`).Scan(&tmpl.ID, &tmpl.TargetRole, &tmpl.Source, &tmpl.Version, &tmpl.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("roadmap_repo: get curated template: %w", err)
	}
	return &tmpl, nil
}

func (r *RoadmapRepo) GetTutorialsByTopic(ctx context.Context, topicID string) ([]models.Tutorial, error) {
	query := `
		SELECT id, source, source_url, title, topic_tags, type, difficulty, estimated_minutes, COALESCE(summary, ''), COALESCE(thumbnail_url, '')
		FROM tutorials
		WHERE $1 = ANY(topic_tags) OR $1 = ANY(string_to_array(title, ' '))
		LIMIT 5
	`
	rows, err := r.pool.Query(ctx, query, topicID)
	if err != nil {
		return []models.Tutorial{}, nil
	}
	defer rows.Close()

	var tutorials []models.Tutorial
	for rows.Next() {
		var t models.Tutorial
		var thumb string
		var summary string
		if err := rows.Scan(
			&t.ID, &t.Source, &t.SourceURL, &t.Title, &t.TopicTags,
			&t.Type, &t.Difficulty, &t.EstimatedMinutes, &summary, &thumb,
		); err == nil {
			t.Summary = summary
			t.ThumbnailURL = thumb
			t.Status = "UNREAD"
			tutorials = append(tutorials, t)
		}
	}
	return tutorials, nil
}

func (r *RoadmapRepo) GetTutorialsByIDs(ctx context.Context, ids []uuid.UUID) ([]models.Tutorial, error) {
	if len(ids) == 0 {
		return []models.Tutorial{}, nil
	}
	query := `
		SELECT id, source, source_url, title, topic_tags, type, difficulty, estimated_minutes, COALESCE(summary, ''), COALESCE(thumbnail_url, '')
		FROM tutorials
		WHERE id = ANY($1)
	`
	rows, err := r.pool.Query(ctx, query, ids)
	if err != nil {
		return []models.Tutorial{}, nil
	}
	defer rows.Close()

	var tutorials []models.Tutorial
	for rows.Next() {
		var t models.Tutorial
		var thumb string
		var summary string
		if err := rows.Scan(
			&t.ID, &t.Source, &t.SourceURL, &t.Title, &t.TopicTags,
			&t.Type, &t.Difficulty, &t.EstimatedMinutes, &summary, &thumb,
		); err == nil {
			t.Summary = summary
			t.ThumbnailURL = thumb
			t.Status = "UNREAD"
			tutorials = append(tutorials, t)
		}
	}
	return tutorials, nil
}

