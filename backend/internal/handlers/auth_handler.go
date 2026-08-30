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
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"

	"transverse/internal/cache"
	"transverse/internal/config"
	"transverse/internal/middleware"
	"transverse/internal/oauth"
	"transverse/internal/repository"
)

type AuthHandler struct {
	cfg       *config.Config
	oauthRepo *repository.OAuthRepo
	userRepo  *repository.UserRepo
	cache     cache.Cache
}

func NewAuthHandler(cfg *config.Config, oauthRepo *repository.OAuthRepo, userRepo *repository.UserRepo, cache cache.Cache) *AuthHandler {
	return &AuthHandler{
		cfg:       cfg,
		oauthRepo: oauthRepo,
		userRepo:  userRepo,
		cache:     cache,
	}
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Username string `json:"username,omitempty"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Register creates a new user account with email and password (no email verification needed).
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	req.Password = strings.TrimSpace(req.Password)
	req.Username = strings.TrimSpace(req.Username)

	if req.Email == "" || !strings.Contains(req.Email, "@") {
		writeError(w, http.StatusBadRequest, "A valid email address is required")
		return
	}
	if len(req.Password) < 6 {
		writeError(w, http.StatusBadRequest, "Password must be at least 6 characters long")
		return
	}

	if req.Username == "" {
		parts := strings.Split(req.Email, "@")
		req.Username = parts[0]
	}

	existing, _ := h.userRepo.GetByEmailOrUsername(r.Context(), req.Email)
	if existing != nil {
		writeError(w, http.StatusConflict, "An account with this email or username already exists")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	userID := uuid.New().String()
	user, err := h.userRepo.CreateWithPassword(r.Context(), userID, req.Username, req.Email, string(hash))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create user account")
		return
	}

	tokens, err := h.mintTokens(r.Context(), user.ID, user.Username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to generate tokens")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"access_token":  tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
		"expires_in":    tokens.ExpiresIn,
		"user":          user,
	})
}

// Login authenticates a user with email and password.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	req.Password = strings.TrimSpace(req.Password)

	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "Email and password are required")
		return
	}

	user, err := h.userRepo.GetByEmailOrUsername(r.Context(), req.Email)
	if err != nil || user == nil {
		writeError(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	if user.PasswordHash == "" {
		writeError(w, http.StatusUnauthorized, "Account has no password set. Please log in using your OAuth provider or register.")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	tokens, err := h.mintTokens(r.Context(), user.ID, user.Username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to generate tokens")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"access_token":  tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
		"expires_in":    tokens.ExpiresIn,
		"user":          user,
	})
}

func (h *AuthHandler) getConfig(provider string) (*oauth2.Config, error) {
	switch provider {
	case "github":
		return oauth.NewGithubConfig(h.cfg), nil
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

	// Browser-facing leg of the OAuth dance: redirect to the frontend rather than
	// writing a raw JSON body, which the browser would otherwise land on directly.
	h.issueTokensRedirect(w, r, userID, username)
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

	// POST /auth/refresh is called programmatically by the frontend and must keep
	// returning a JSON body (never a redirect).
	h.issueTokensJSON(w, r.Context(), user.ID, user.Username)
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

// tokenPair holds a freshly minted access/refresh token pair.
type tokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
}

// mintTokens creates a new access/refresh token pair for the given user and persists
// the refresh token hash. It performs no HTTP writes — callers decide how to deliver
// the tokens (JSON body vs. redirect), which is exactly why this is split out from the
// old issueTokens: the OAuth callback needs a very different delivery mechanism than
// POST /auth/refresh, but both need identical minting logic.
func (h *AuthHandler) mintTokens(ctx context.Context, userID, username string) (*tokenPair, error) {
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
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	// Refresh Token
	rtBytes := make([]byte, 32)
	rand.Read(rtBytes)
	refreshToken := base64.URLEncoding.EncodeToString(rtBytes)

	hash := sha256.Sum256([]byte(refreshToken))
	tokenHash := hex.EncodeToString(hash[:])

	rtExp := time.Now().Add(time.Duration(h.cfg.JWTRefreshTTLDays) * 24 * time.Hour)
	if err := h.oauthRepo.CreateRefreshToken(ctx, userID, tokenHash, rtExp); err != nil {
		return nil, fmt.Errorf("failed to save refresh token: %w", err)
	}

	return &tokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(time.Until(exp).Seconds()),
	}, nil
}

// issueTokensJSON mints a token pair and writes it as the JSON response body:
// {access_token, refresh_token, expires_in}. Used by POST /auth/refresh, which is
// called programmatically by the frontend and expects a JSON body, not a redirect.
func (h *AuthHandler) issueTokensJSON(w http.ResponseWriter, ctx context.Context, userID, username string) {
	tokens, err := h.mintTokens(ctx, userID, username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"access_token":  tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
		"expires_in":    tokens.ExpiresIn,
	})
}

// issueTokensRedirect mints a token pair and 302-redirects the browser to the
// frontend's OAuth callback route, carrying the tokens as query parameters so the SPA
// can pick them up and store them. Used only by the OAuth provider callback: the
// browser is sitting on this backend URL after bouncing through GitHub/Google, and a
// raw JSON body would leave it stranded on an API response instead of back in the app.
//
// NOTE: passing tokens via query string is acceptable for this hackathon timeline but
// is not hardened — a short-lived, single-use exchange code (swapped for tokens via a
// follow-up POST from the frontend) would be the hardening step, avoiding bearer
// tokens landing in browser history / Referer headers / server access logs.
func (h *AuthHandler) issueTokensRedirect(w http.ResponseWriter, r *http.Request, userID, username string) {
	tokens, err := h.mintTokens(r.Context(), userID, username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	redirectURL := fmt.Sprintf("%s/auth/callback?access_token=%s&refresh_token=%s&expires_in=%d",
		strings.TrimRight(h.cfg.FrontendOrigin, "/"),
		url.QueryEscape(tokens.AccessToken),
		url.QueryEscape(tokens.RefreshToken),
		tokens.ExpiresIn,
	)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}
