package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"velocity/internal/models"
)

type SessionRepo struct {
	pool *pgxpool.Pool
}

func NewSessionRepo(pool *pgxpool.Pool) *SessionRepo {
	return &SessionRepo{pool: pool}
}

type SessionWithUser struct {
	models.LearnSession
	UserEmail string
}

func (r *SessionRepo) Create(ctx context.Context, s *models.LearnSession) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO learn_sessions (id, user_id, mode, chapter, scope, theta_start, theta_current,
			question_count, current_question_id, status, question_limit, biometric_enabled,
			question_ordering, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7, $8, $9, $10, $11, $12, $13, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, s.ID, s.UserID, s.Mode, s.Chapter, s.ScopeRaw, s.ThetaStart, s.ThetaCurrent,
		s.QuestionCount, s.CurrentQuestionID, s.Status, s.QuestionLimit, s.BiometricEnabled,
		s.QuestionOrdering)
	if err != nil {
		return fmt.Errorf("session_repo: create: %w", err)
	}
	return nil
}

func (r *SessionRepo) GetByID(ctx context.Context, id string) (*models.LearnSession, error) {
	var s models.LearnSession
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, mode, chapter, scope, theta_start, theta_current,
		       question_count, current_question_id, responses, status, question_limit,
		       biometric_enabled, biometric_logs, biometric_baseline, question_ordering, created_at, updated_at
		FROM learn_sessions WHERE id = $1
	`, id).Scan(
		&s.ID, &s.UserID, &s.Mode, &s.Chapter, &s.ScopeRaw,
		&s.ThetaStart, &s.ThetaCurrent, &s.QuestionCount, &s.CurrentQuestionID,
		&s.ResponsesRaw, &s.Status, &s.QuestionLimit, &s.BiometricEnabled,
		&s.BiometricLogsRaw, &s.BiometricBaselineRaw, &s.QuestionOrdering, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("session_repo: get by id %q: %w", id, err)
	}
	return &s, nil
}

func (r *SessionRepo) GetAllSessions(ctx context.Context) ([]SessionWithUser, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT s.id, s.user_id, u.email, s.mode, s.chapter, s.theta_start, s.theta_current,
		       s.question_count, s.current_question_id, s.responses, s.status, s.created_at, s.updated_at
		FROM learn_sessions s
		JOIN users u ON u.id = s.user_id
		ORDER BY s.created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("session_repo: get all sessions: %w", err)
	}
	defer rows.Close()

	var result []SessionWithUser
	for rows.Next() {
		var swu SessionWithUser
		if err := rows.Scan(
			&swu.ID, &swu.UserID, &swu.UserEmail, &swu.Mode, &swu.Chapter,
			&swu.ThetaStart, &swu.ThetaCurrent, &swu.QuestionCount, &swu.CurrentQuestionID,
			&swu.ResponsesRaw, &swu.Status, &swu.CreatedAt, &swu.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("session_repo: scan session: %w", err)
		}
		result = append(result, swu)
	}
	return result, rows.Err()
}

func (r *SessionRepo) GetActiveByUserAndChapter(ctx context.Context, userID, chapter string) (*models.LearnSession, error) {
	var s models.LearnSession
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, mode, chapter, scope, theta_start, theta_current,
		       question_count, current_question_id, responses, status, question_limit,
		       biometric_enabled, biometric_logs, biometric_baseline, question_ordering, created_at, updated_at
		FROM learn_sessions
		WHERE user_id = $1 AND chapter = $2 AND status = 'ACTIVE'
		ORDER BY created_at DESC LIMIT 1
	`, userID, chapter).Scan(
		&s.ID, &s.UserID, &s.Mode, &s.Chapter, &s.ScopeRaw,
		&s.ThetaStart, &s.ThetaCurrent, &s.QuestionCount, &s.CurrentQuestionID,
		&s.ResponsesRaw, &s.Status, &s.QuestionLimit, &s.BiometricEnabled,
		&s.BiometricLogsRaw, &s.BiometricBaselineRaw, &s.QuestionOrdering, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("session_repo: get active by user and chapter: %w", err)
	}
	return &s, nil
}

func (r *SessionRepo) GetLastCompletedByUser(ctx context.Context, userID string) (*models.LearnSession, error) {
	var s models.LearnSession
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, mode, chapter, scope, theta_start, theta_current,
		       question_count, current_question_id, responses, status, question_limit,
		       biometric_enabled, biometric_logs, biometric_baseline, question_ordering, created_at, updated_at
		FROM learn_sessions
		WHERE user_id = $1 AND status = 'COMPLETED'
		ORDER BY updated_at DESC LIMIT 1
	`, userID).Scan(
		&s.ID, &s.UserID, &s.Mode, &s.Chapter, &s.ScopeRaw,
		&s.ThetaStart, &s.ThetaCurrent, &s.QuestionCount, &s.CurrentQuestionID,
		&s.ResponsesRaw, &s.Status, &s.QuestionLimit, &s.BiometricEnabled,
		&s.BiometricLogsRaw, &s.BiometricBaselineRaw, &s.QuestionOrdering, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("session_repo: get last completed: %w", err)
	}
	return &s, nil
}

func (r *SessionRepo) GetAllByUser(ctx context.Context, userID string) ([]*models.LearnSession, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, mode, chapter, scope, theta_start, theta_current,
		       question_count, current_question_id, responses, status, question_limit,
		       biometric_enabled, biometric_logs, biometric_baseline, question_ordering, created_at, updated_at
		FROM learn_sessions
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("session_repo: get all by user: %w", err)
	}
	defer rows.Close()

	var result []*models.LearnSession
	for rows.Next() {
		var s models.LearnSession
		if err := rows.Scan(
			&s.ID, &s.UserID, &s.Mode, &s.Chapter, &s.ScopeRaw,
			&s.ThetaStart, &s.ThetaCurrent, &s.QuestionCount, &s.CurrentQuestionID,
			&s.ResponsesRaw, &s.Status, &s.QuestionLimit, &s.BiometricEnabled,
			&s.BiometricLogsRaw, &s.BiometricBaselineRaw, &s.QuestionOrdering, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("session_repo: scan session: %w", err)
		}
		result = append(result, &s)
	}
	return result, rows.Err()
}

func (r *SessionRepo) Abandon(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `UPDATE learn_sessions SET status = 'ABANDONED', updated_at = CURRENT_TIMESTAMP WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("session_repo: abandon %q: %w", id, err)
	}
	return nil
}

func (r *SessionRepo) AbandonStale(ctx context.Context, staleAge time.Duration) (int, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE learn_sessions SET status = 'ABANDONED', updated_at = CURRENT_TIMESTAMP
		WHERE status = 'ACTIVE' AND updated_at < $1
	`, time.Now().Add(-staleAge))
	if err != nil {
		return 0, fmt.Errorf("session_repo: abandon stale: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

func (r *SessionRepo) LockSessionForUpdate(ctx context.Context, tx pgx.Tx, id string) error {
	_, err := tx.Exec(ctx, `SELECT 1 FROM learn_sessions WHERE id = $1 FOR UPDATE`, id)
	if err != nil {
		return fmt.Errorf("session_repo: lock session: %w", err)
	}
	return nil
}

func (r *SessionRepo) AppendResponseAndUpdateMetadataTx(ctx context.Context, tx pgx.Tx, id string, resp models.SessionResponse, thetaCurrent float32, questionCount int, nextQID *string) error {
	respJSON, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("session_repo: marshal response: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE learn_sessions
		SET responses = COALESCE(responses, '[]'::jsonb) || $2::jsonb,
		    theta_current = $3, question_count = $4,
		    current_question_id = $5, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`, id, respJSON, thetaCurrent, questionCount, nextQID)
	if err != nil {
		return fmt.Errorf("session_repo: append response: %w", err)
	}
	return nil
}

func (r *SessionRepo) AppendBiometricLogs(ctx context.Context, id string, snapshots []models.BiometricSnapshot) error {
	data, err := json.Marshal(snapshots)
	if err != nil {
		return fmt.Errorf("session_repo: marshal snapshots: %w", err)
	}
	_, err = r.pool.Exec(ctx, `
		UPDATE learn_sessions
		SET biometric_logs = COALESCE(biometric_logs, '[]'::jsonb) || $2::jsonb,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`, id, data)
	if err != nil {
		return fmt.Errorf("session_repo: append biometric logs: %w", err)
	}
	return nil
}

func (r *SessionRepo) SetBiometricBaseline(ctx context.Context, id string, baseline models.BiometricBaseline) error {
	data, err := json.Marshal(baseline)
	if err != nil {
		return fmt.Errorf("session_repo: marshal baseline: %w", err)
	}
	_, err = r.pool.Exec(ctx, `
		UPDATE learn_sessions SET biometric_baseline = $2::jsonb, updated_at = CURRENT_TIMESTAMP WHERE id = $1
	`, id, data)
	if err != nil {
		return fmt.Errorf("session_repo: set biometric baseline: %w", err)
	}
	return nil
}

func (r *SessionRepo) SetBiometricMetadata(ctx context.Context, id string, dna models.BiometricDNA, enabled bool) error {
	dnaJSON, err := json.Marshal(dna)
	if err != nil {
		return fmt.Errorf("session_repo: marshal dna: %w", err)
	}
	_, err = r.pool.Exec(ctx, `
		UPDATE learn_sessions SET biometric_dna_snapshot = $2::jsonb, biometric_enabled = $3, updated_at = CURRENT_TIMESTAMP WHERE id = $1
	`, id, dnaJSON, enabled)
	if err != nil {
		return fmt.Errorf("session_repo: set biometric metadata: %w", err)
	}
	return nil
}

func (r *SessionRepo) CloseTx(ctx context.Context, tx pgx.Tx, id string, thetaFinal float32) error {
	_, err := tx.Exec(ctx, `
		UPDATE learn_sessions
		SET status = 'COMPLETED', theta_current = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`, id, thetaFinal)
	if err != nil {
		return fmt.Errorf("session_repo: close session: %w", err)
	}
	return nil
}

func (r *SessionRepo) GetStaleActiveSessions(ctx context.Context, cutoff time.Time) ([]*models.LearnSession, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, mode, chapter, scope, theta_start, theta_current,
		       question_count, current_question_id, responses, status, question_limit,
		       biometric_enabled, biometric_logs, biometric_baseline, question_ordering, created_at, updated_at
		FROM learn_sessions
		WHERE status = 'ACTIVE' AND updated_at < $1
		ORDER BY updated_at ASC
	`, cutoff)
	if err != nil {
		return nil, fmt.Errorf("session_repo: get stale active: %w", err)
	}
	defer rows.Close()

	var result []*models.LearnSession
	for rows.Next() {
		var s models.LearnSession
		if err := rows.Scan(
			&s.ID, &s.UserID, &s.Mode, &s.Chapter, &s.ScopeRaw,
			&s.ThetaStart, &s.ThetaCurrent, &s.QuestionCount, &s.CurrentQuestionID,
			&s.ResponsesRaw, &s.Status, &s.QuestionLimit, &s.BiometricEnabled,
			&s.BiometricLogsRaw, &s.BiometricBaselineRaw, &s.QuestionOrdering, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("session_repo: scan stale session: %w", err)
		}
		result = append(result, &s)
	}
	return result, rows.Err()
}

func (r *SessionRepo) GetActiveByUser(ctx context.Context, userID string) (*models.LearnSession, error) {
	var s models.LearnSession
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, mode, chapter, scope, theta_start, theta_current,
		       question_count, current_question_id, responses, status, question_limit,
		       biometric_enabled, biometric_logs, biometric_baseline, question_ordering, created_at, updated_at
		FROM learn_sessions
		WHERE user_id = $1 AND status = 'ACTIVE'
		ORDER BY updated_at DESC LIMIT 1
	`, userID).Scan(
		&s.ID, &s.UserID, &s.Mode, &s.Chapter, &s.ScopeRaw,
		&s.ThetaStart, &s.ThetaCurrent, &s.QuestionCount, &s.CurrentQuestionID,
		&s.ResponsesRaw, &s.Status, &s.QuestionLimit, &s.BiometricEnabled,
		&s.BiometricLogsRaw, &s.BiometricBaselineRaw, &s.QuestionOrdering, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("session_repo: get active by user: %w", err)
	}
	return &s, nil
}

func (r *SessionRepo) UpdateCurrentQuestion(ctx context.Context, id, questionID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE learn_sessions SET current_question_id = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1
	`, id, questionID)
	if err != nil {
		return fmt.Errorf("session_repo: update current question: %w", err)
	}
	return nil
}
