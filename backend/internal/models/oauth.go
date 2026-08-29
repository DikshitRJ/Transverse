package models

import (
	"time"
)

// OAuthAccount represents an external identity linked to a User.
type OAuthAccount struct {
	ID             string    `json:"id" db:"id"`
	UserID         string    `json:"user_id" db:"user_id"`
	Provider       string    `json:"provider" db:"provider"`
	ProviderUserID string    `json:"provider_user_id" db:"provider_user_id"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// RefreshToken represents a long-lived token used to obtain new access tokens.
type RefreshToken struct {
	ID        string     `json:"id" db:"id"`
	UserID    string     `json:"user_id" db:"user_id"`
	TokenHash string     `json:"token_hash" db:"token_hash"`
	ExpiresAt time.Time  `json:"expires_at" db:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at" db:"revoked_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}
