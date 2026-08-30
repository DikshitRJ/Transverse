package handlers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/oauth2"

	"transverse/internal/cache"
	"transverse/internal/config"
	"transverse/internal/repository"
	"transverse/internal/middleware"
	"transverse/internal/oauth"
)

type AuthHandler struct {
	cfg        *config.Config
	oauthRepo  *repository.OAuthRepo
	userRepo   *repository.UserRepo
	cache      cache.Cache
}

func NewAuthHandler(cfg *config.Config, oauthRepo *repository.OAuthRepo, userRepo *repository.UserRepo, cache cache.Cache) *AuthHandler {
	return &AuthHandler{
		cfg:       cfg,
		oauthRepo: oauthRepo,
		userRepo:  userRepo,
		cache:     cache,
	}
}

func (h *AuthHandler) getConfig(provider string) (*oauth2.Config, error) {
	switch provider {
	case "github":
		return oauth.NewGithubConfig(h.cfg), nil
	case "google":
		return oauth.NewGoogleConfig(h.cfg), nil
	default:
		return nil, fmt.Errorf("unsupported provider")
	}
}

func (h *AuthHandler) Redirect(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	conf, err := h.getConfig(provider)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	stateBytes := make([]byte, 16)
	rand.Read(stateBytes)
	state := hex.EncodeToString(stateBytes)

	// Set state in a secure HttpOnly cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		MaxAge:   300,
		HttpOnly: true,
		Secure:   r.TLS != nil,
	})

	url := conf.AuthCodeURL(state)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (h *AuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	state := r.FormValue("state")
	code := r.FormValue("code")

	cookie, err := r.Cookie("oauth_state")
	if err != nil || cookie.Value != state {
		writeError(w, http.StatusBadRequest, "State mismatch or cookie expired")
		return
	}

	conf, err := h.getConfig(provider)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	token, err := conf.Exchange(r.Context(), code)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Failed to exchange token")
		return
	}

	var providerUserID, username, email string
	if provider == "github" {
		client := conf.Client(r.Context(), token)
		resp, err := client.Get(h.cfg.GithubAPIBase + "/user")
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to fetch user data")
			return
		}
		defer resp.Body.Close()
		var u struct {
			ID    int    `json:"id"`
			Login string `json:"login"`
			Email string `json:"email"`
		}
		json.NewDecoder(resp.Body).Decode(&u)
		providerUserID = fmt.Sprintf("%d", u.ID)
		username = u.Login
		email = u.Email
	} else if provider == "google" {
		client := conf.Client(r.Context(), token)
		resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to fetch user data")
			return
		}
		defer resp.Body.Close()
		var u struct {
			ID    string `json:"id"`
			Email string `json:"email"`
			Name  string `json:"name"`
		}
		json.NewDecoder(resp.Body).Decode(&u)
		providerUserID = u.ID
		username = u.Name
		email = u.Email
	}

	if providerUserID == "" {
		writeError(w, http.StatusInternalServerError, "Provider did not return a user ID")
		return
	}

	// 1. Get or create OAuth account and User
	acc, err := h.oauthRepo.GetAccountByProvider(r.Context(), provider, providerUserID)
	var userID string
	if err != nil {
		// Create new user (or you could match by email if it exists)
		newID := uuid.New().String()
		user, err := h.userRepo.GetOrCreate(r.Context(), newID, username, email)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to create user")
			return
		}
		userID = user.ID
		err = h.oauthRepo.LinkAccount(r.Context(), userID, provider, providerUserID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to link oauth account")
			return
		}
	} else {
		userID = acc.UserID
	}

	h.issueTokens(w, r.Context(), userID, username)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	// Hash the token to find in DB
	hash := sha256.Sum256([]byte(req.RefreshToken))
	tokenHash := hex.EncodeToString(hash[:])

	rt, err := h.oauthRepo.GetRefreshToken(r.Context(), tokenHash)
	if err != nil || rt.RevokedAt != nil || time.Now().After(rt.ExpiresAt) {
		writeError(w, http.StatusUnauthorized, "Refresh token is invalid or expired")
		return
	}

	// Revoke old token
	_ = h.oauthRepo.RevokeRefreshToken(r.Context(), tokenHash)

	// Issue new ones
	user, err := h.userRepo.GetByID(r.Context(), rt.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "User not found")
		return
	}

	h.issueTokens(w, r.Context(), user.ID, user.Username)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	if req.RefreshToken != "" {
		hash := sha256.Sum256([]byte(req.RefreshToken))
		tokenHash := hex.EncodeToString(hash[:])
		_ = h.oauthRepo.RevokeRefreshToken(r.Context(), tokenHash)
	}

	// Denylist the current access token
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		token, _ := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) { return []byte(h.cfg.JWTSecret), nil })
		if token != nil {
			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				if jti, ok := claims["jti"].(string); ok && jti != "" {
					if exp, ok := claims["exp"].(float64); ok {
						ttl := time.Until(time.Unix(int64(exp), 0))
						if ttl > 0 {
							_ = h.cache.Set(r.Context(), "jwt:denylist:"+jti, true, ttl)
						}
					}
				}
			}
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "User ID not found in context")
		return
	}

	user, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusNotFound, "User not found")
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func (h *AuthHandler) issueTokens(w http.ResponseWriter, ctx context.Context, userID, username string) {
	// Access Token
	jti := uuid.New().String()
	exp := time.Now().Add(time.Duration(h.cfg.JWTAccessTTLMinutes) * time.Minute)
	claims := jwt.MapClaims{
		"sub":      userID,
		"username": username,
		"jti":      jti,
		"exp":      exp.Unix(),
		"iat":      time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := token.SignedString([]byte(h.cfg.JWTSecret))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to sign access token")
		return
	}

	// Refresh Token
	rtBytes := make([]byte, 32)
	rand.Read(rtBytes)
	refreshToken := base64.URLEncoding.EncodeToString(rtBytes)

	hash := sha256.Sum256([]byte(refreshToken))
	tokenHash := hex.EncodeToString(hash[:])

	rtExp := time.Now().Add(time.Duration(h.cfg.JWTRefreshTTLDays) * 24 * time.Hour)
	err = h.oauthRepo.CreateRefreshToken(ctx, userID, tokenHash, rtExp)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save refresh token")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    int(time.Until(exp).Seconds()),
	})
}
