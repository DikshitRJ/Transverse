package handlers

import (
	"log/slog"
	"net/http"
	"strconv"

	"transverse/internal/middleware"
	"transverse/internal/models"
	"transverse/internal/repository"
)

// UserHandler handles endpoints related to user profiles, analytics, and historical sessions.
type UserHandler struct {
	userRepo    *repository.UserRepo
	statsRepo   *repository.StatsRepo
	sessionRepo *repository.SessionRepo
}

// NewUserHandler constructs a new UserHandler.
func NewUserHandler(userRepo *repository.UserRepo, statsRepo *repository.StatsRepo, sessionRepo *repository.SessionRepo) *UserHandler {
	return &UserHandler{
		userRepo:    userRepo,
		statsRepo:   statsRepo,
		sessionRepo: sessionRepo,
	}
}

// GetProfile retrieves the authenticated user's psychometrics, rating, and LearningDNA.
// GET /api/v1/user/profile
func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok || userID == "" {
		userID = "dev-user-001"
	}

	user, err := h.userRepo.GetOrCreate(r.Context(), userID, "", "")
	if err != nil {
		slog.Error("failed to retrieve user profile", "user_id", userID, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to retrieve profile")
		return
	}

	dna, err := user.DNA()
	if err != nil {
		dna = models.DefaultDNA()
	}

	resp := models.UserProfileResponse{
		ID:           user.ID,
		Username:     user.Username,
		Email:        user.Email,
		Theta:        user.Theta,
		GlickoRating: user.GlickoRating,
		GlickoRD:     user.GlickoRD,
		DNA:          dna,
		CreatedAt:    user.CreatedAt,
	}

	writeJSON(w, http.StatusOK, resp)
}

// GetHistory retrieves paginated historical practice sessions for the authenticated user.
// GET /api/v1/user/history?limit=10&offset=0
func (h *UserHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok || userID == "" {
		userID = "dev-user-001"
	}

	limit := 10
	offset := 0

	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}
	if oStr := r.URL.Query().Get("offset"); oStr != "" {
		if o, err := strconv.Atoi(oStr); err == nil && o >= 0 {
			offset = o
		}
	}

	sessions, err := h.sessionRepo.GetHistoryByUser(r.Context(), userID, limit, offset)
	if err != nil {
		slog.Error("failed to retrieve session history", "user_id", userID, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to retrieve session history")
		return
	}

	if sessions == nil {
		sessions = []models.PracticeSession{}
	}

	writeJSON(w, http.StatusOK, sessions)
}
