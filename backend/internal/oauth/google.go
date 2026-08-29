package oauth

import (
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"transverse/internal/config"
)

func NewGoogleConfig(cfg *config.Config) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     cfg.OAuthGoogleClientID,
		ClientSecret: cfg.OAuthGoogleClientSecret,
		RedirectURL:  cfg.OAuthGoogleRedirectURL,
		Scopes:       []string{"openid", "profile", "email"},
		Endpoint:     google.Endpoint,
	}
}
