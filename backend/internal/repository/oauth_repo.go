package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"transverse/internal/models"
)

type OAuthRepo struct {
	pool *pgxpool.Pool
}

func NewOAuthRepo(pool *pgxpool.Pool) *OAuthRepo {
	return &OAuthRepo{pool: pool}
}

func (r *OAuthRepo) GetAccountByProvider(ctx context.Context, provider, providerUserID string) (*models.OAuthAccount, error) {
	var acc models.OAuthAccount
	err := r.pool.QueryRow(ctx, 
		"SELECT id, user_id, provider, provider_user_id, created_at FROM oauth_accounts WHERE provider = $1 AND provider_user_id = $2",
		provider, providerUserID).Scan(&acc.ID, &acc.UserID, &acc.Provider, &acc.ProviderUserID, &acc.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &acc, nil
}

func (r *OAuthRepo) LinkAccount(ctx context.Context, userID, provider, providerUserID string) error {
	_, err := r.pool.Exec(ctx,
		"INSERT INTO oauth_accounts (user_id, provider, provider_user_id) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING",
		userID, provider, providerUserID)
	return err
}

func (r *OAuthRepo) CreateRefreshToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		"INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)",
		userID, tokenHash, expiresAt)
	return err
}

func (r *OAuthRepo) GetRefreshToken(ctx context.Context, tokenHash string) (*models.RefreshToken, error) {
	var rt models.RefreshToken
	err := r.pool.QueryRow(ctx,
		"SELECT id, user_id, token_hash, expires_at, revoked_at, created_at FROM refresh_tokens WHERE token_hash = $1",
		tokenHash).Scan(&rt.ID, &rt.UserID, &rt.TokenHash, &rt.ExpiresAt, &rt.RevokedAt, &rt.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &rt, nil
}

func (r *OAuthRepo) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE refresh_tokens SET revoked_at = NOW() WHERE token_hash = $1", tokenHash)
	return err
}
