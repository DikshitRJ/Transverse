package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"transverse/internal/cache"
	"transverse/internal/models"
)

// UserRepo manages database access and caching for User profiles and psychometric DNA.
type UserRepo struct {
	pool  *pgxpool.Pool
	cache cache.Cache
}

// NewUserRepo constructs a new UserRepo instance.
func NewUserRepo(pool *pgxpool.Pool, c cache.Cache) *UserRepo {
	return &UserRepo{
		pool:  pool,
		cache: c,
	}
}

// GetByID retrieves a user by ID, checking cache first.
func (r *UserRepo) GetByID(ctx context.Context, id string) (*models.User, error) {
	cacheKey := fmt.Sprintf("user:%s", id)
	if r.cache != nil {
		var cached models.User
		if err := r.cache.Get(ctx, cacheKey, &cached); err == nil {
			return &cached, nil
		}
	}

	var u models.User
	err := r.pool.QueryRow(ctx, `
		SELECT id, username, email, theta, glicko_rating, glicko_rd, glicko_vol,
		       dna, created_at, updated_at
		FROM users WHERE id = $1
	`, id).Scan(
		&u.ID, &u.Username, &u.Email, &u.Theta, &u.GlickoRating, &u.GlickoRD, &u.GlickoVol,
		&u.DNARaw, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found: %s", id)
		}
		return nil, fmt.Errorf("user_repo: get by id %q: %w", id, err)
	}

	if r.cache != nil {
		_ = r.cache.Set(ctx, cacheKey, u, 60*time.Second)
	}

	return &u, nil
}

// GetOrCreate retrieves an existing user by ID or creates a baseline user account if not found.
func (r *UserRepo) GetOrCreate(ctx context.Context, id, username, email string) (*models.User, error) {
	u, err := r.GetByID(ctx, id)
	if err == nil && u != nil {
		return u, nil
	}

	defaultDNABytes, err := json.Marshal(models.DefaultDNA())
	if err != nil {
		return nil, fmt.Errorf("user_repo: marshal default dna: %w", err)
	}

	if username == "" {
		username = "user_" + id
	}
	if email == "" {
		email = username + "@transverse.local"
	}

	var created models.User
	err = r.pool.QueryRow(ctx, `
		INSERT INTO users (id, username, email, theta, glicko_rating, glicko_rd, glicko_vol, dna, created_at, updated_at)
		VALUES ($1, $2, $3, 1500, 1500, 350, 0.06, $4::jsonb, NOW(), NOW())
		ON CONFLICT (id) DO UPDATE SET updated_at = NOW()
		RETURNING id, username, email, theta, glicko_rating, glicko_rd, glicko_vol, dna, created_at, updated_at
	`, id, username, email, defaultDNABytes).Scan(
		&created.ID, &created.Username, &created.Email, &created.Theta, &created.GlickoRating,
		&created.GlickoRD, &created.GlickoVol, &created.DNARaw, &created.CreatedAt, &created.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("user_repo: get or create: %w", err)
	}

	if r.cache != nil {
		_ = r.cache.Set(ctx, fmt.Sprintf("user:%s", created.ID), created, 60*time.Second)
	}

	return &created, nil
}

// Update updates a user's latent theta, Glicko rating parameters, and profile fields.
func (r *UserRepo) Update(ctx context.Context, u *models.User) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE users
		SET username = $2,
		    email = $3,
		    theta = $4,
		    glicko_rating = $5,
		    glicko_rd = $6,
		    glicko_vol = $7,
		    dna = $8::jsonb,
		    updated_at = NOW()
		WHERE id = $1
	`, u.ID, u.Username, u.Email, u.Theta, u.GlickoRating, u.GlickoRD, u.GlickoVol, u.DNARaw)
	if err != nil {
		return fmt.Errorf("user_repo: update: %w", err)
	}

	if r.cache != nil {
		_ = r.cache.Del(ctx, fmt.Sprintf("user:%s", u.ID))
		_ = r.cache.Del(ctx, fmt.Sprintf("dna:%s", u.ID))
	}

	return nil
}

// UpdateDNA updates only the psychometric DNA JSONB field of a user.
func (r *UserRepo) UpdateDNA(ctx context.Context, id string, dna models.LearningDNA) error {
	dnaBytes, err := json.Marshal(dna)
	if err != nil {
		return fmt.Errorf("user_repo: marshal dna: %w", err)
	}

	_, err = r.pool.Exec(ctx, `
		UPDATE users
		SET dna = $2::jsonb,
		    updated_at = NOW()
		WHERE id = $1
	`, id, dnaBytes)
	if err != nil {
		return fmt.Errorf("user_repo: update dna: %w", err)
	}

	if r.cache != nil {
		_ = r.cache.Del(ctx, fmt.Sprintf("user:%s", id))
		_ = r.cache.Del(ctx, fmt.Sprintf("dna:%s", id))
	}

	return nil
}
