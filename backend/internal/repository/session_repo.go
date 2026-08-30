package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"transverse/internal/models"
)

// SessionRepo handles persistence and atomic state transitions for PracticeSession entities.
type SessionRepo struct {
	pool *pgxpool.Pool
}

// NewSessionRepo constructs a new SessionRepo.
func NewSessionRepo(pool *pgxpool.Pool) *SessionRepo {
	return &SessionRepo{pool: pool}
}

// Create inserts a new practice session record.
func (r *SessionRepo) Create(ctx context.Context, s *models.PracticeSession) error {
	scopeBytes := s.ScopeRaw
	if len(scopeBytes) == 0 {
		scopeBytes = []byte("{}")
	}

	responsesBytes := s.ResponsesRaw
	if len(responsesBytes) == 0 {
		responsesBytes = []byte("[]")
	}

	_, err := r.pool.Exec(ctx, `
		INSERT INTO practice_sessions (
			id, user_id, mode, scope, theta_start, theta_current,
			current_problem_id, responses, question_count, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4::jsonb, $5, $6, $7, $8::jsonb, $9, $10, NOW(), NOW())
	`,
		s.ID, s.UserID, s.Mode, scopeBytes, s.ThetaStart, s.ThetaCurrent,
		s.CurrentProblemID, responsesBytes, s.QuestionCount, s.Status,
	)
	if err != nil {
		return fmt.Errorf("session_repo: create: %w", err)
	}

	return nil
}

// GetByID retrieves a single session by its unique ID.
func (r *SessionRepo) GetByID(ctx context.Context, id string) (*models.PracticeSession, error) {
	var s models.PracticeSession
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, mode, scope, theta_start, theta_current,
		       current_problem_id, responses, question_count, status, created_at, updated_at
		FROM practice_sessions WHERE id = $1
	`, id).Scan(
		&s.ID, &s.UserID, &s.Mode, &s.ScopeRaw, &s.ThetaStart, &s.ThetaCurrent,
		&s.CurrentProblemID, &s.ResponsesRaw, &s.QuestionCount, &s.Status, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("session not found: %s", id)
		}
		return nil, fmt.Errorf("session_repo: get by id %q: %w", id, err)
	}

	return &s, nil
}

// GetActiveByUser retrieves the current ACTIVE practice session for a user, or nil if none exists.
func (r *SessionRepo) GetActiveByUser(ctx context.Context, userID string) (*models.PracticeSession, error) {
	var s models.PracticeSession
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, mode, scope, theta_start, theta_current,
		       current_problem_id, responses, question_count, status, created_at, updated_at
		FROM practice_sessions
		WHERE user_id = $1 AND status = 'ACTIVE'
		ORDER BY created_at DESC LIMIT 1
	`, userID).Scan(
		&s.ID, &s.UserID, &s.Mode, &s.ScopeRaw, &s.ThetaStart, &s.ThetaCurrent,
		&s.CurrentProblemID, &s.ResponsesRaw, &s.QuestionCount, &s.Status, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("session_repo: get active by user %q: %w", userID, err)
	}

	return &s, nil
}

// AppendResponse atomically appends a response record to responses JSONB and updates the current problem and theta.
func (r *SessionRepo) AppendResponse(
	ctx context.Context,
	sessionID string,
	resp models.SessionResponse,
	nextProblemID *string,
	newTheta float64,
) error {
	respBytes, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("session_repo: marshal response: %w", err)
	}

	_, err = r.pool.Exec(ctx, `
		UPDATE practice_sessions
		SET responses = responses || $2::jsonb,
		    question_count = question_count + 1,
		    theta_current = $3,
		    current_problem_id = $4,
		    updated_at = NOW()
		WHERE id = $1
	`, sessionID, fmt.Sprintf("[%s]", string(respBytes)), newTheta, nextProblemID)
	if err != nil {
		return fmt.Errorf("session_repo: append response: %w", err)
	}

	return nil
}

// UpdateStatus updates the status string of a practice session.
func (r *SessionRepo) UpdateStatus(ctx context.Context, sessionID string, status string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE practice_sessions
		SET status = $2,
		    updated_at = NOW()
		WHERE id = $1
	`, sessionID, status)
	if err != nil {
		return fmt.Errorf("session_repo: update status: %w", err)
	}
	return nil
}

// CloseSession marks a session as COMPLETED and sets its final theta.
func (r *SessionRepo) CloseSession(ctx context.Context, sessionID string, finalTheta float64) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE practice_sessions
		SET status = 'COMPLETED',
		    theta_current = $2,
		    updated_at = NOW()
		WHERE id = $1
	`, sessionID, finalTheta)
	if err != nil {
		return fmt.Errorf("session_repo: close session: %w", err)
	}
	return nil
}

// GetHistoryByUser returns paginated historical sessions for a user, sorted descending by creation time.
func (r *SessionRepo) GetHistoryByUser(ctx context.Context, userID string, limit, offset int) ([]models.PracticeSession, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, mode, scope, theta_start, theta_current,
		       current_problem_id, responses, question_count, status, created_at, updated_at
		FROM practice_sessions
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("session_repo: get history: %w", err)
	}
	defer rows.Close()

	var sessions []models.PracticeSession
	for rows.Next() {
		var s models.PracticeSession
		if err := rows.Scan(
			&s.ID, &s.UserID, &s.Mode, &s.ScopeRaw, &s.ThetaStart, &s.ThetaCurrent,
			&s.CurrentProblemID, &s.ResponsesRaw, &s.QuestionCount, &s.Status, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("session_repo: scan history session: %w", err)
		}
		sessions = append(sessions, s)
	}

	return sessions, rows.Err()
}

// CleanupStale marks all ACTIVE sessions older than maxAge as ABANDONED.
func (r *SessionRepo) CleanupStale(ctx context.Context, maxAge time.Duration) (int64, error) {
	cutoff := time.Now().Add(-maxAge)
	cmdTag, err := r.pool.Exec(ctx, `
		UPDATE practice_sessions
		SET status = 'ABANDONED',
		    updated_at = NOW()
		WHERE status = 'ACTIVE' AND updated_at < $1
	`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("session_repo: cleanup stale: %w", err)
	}

	return cmdTag.RowsAffected(), nil
}

// RebindUser updates the user_id owner of a session (e.g. claiming a dev-user-001 session upon login).
func (r *SessionRepo) RebindUser(ctx context.Context, sessionID string, newUserID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE practice_sessions
		SET user_id = $2,
		    updated_at = NOW()
		WHERE id = $1
	`, sessionID, newUserID)
	if err != nil {
		return fmt.Errorf("session_repo: rebind user: %w", err)
	}
	return nil
}
