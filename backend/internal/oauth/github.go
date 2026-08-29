package oauth

import (
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"transverse/internal/config"
)

func NewGithubConfig(cfg *config.Config) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     cfg.OAuthGithubClientID,
		ClientSecret: cfg.OAuthGithubClientSecret,
		RedirectURL:  cfg.OAuthGithubRedirectURL,
		Scopes:       []string{"read:user", "user:email"},
		Endpoint:     github.Endpoint,
	}
}
