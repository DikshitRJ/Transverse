package handlers

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"transverse/internal/middleware"
	"transverse/internal/roadmap"
)

// RoadmapHandler exposes HTTP endpoints for the dynamic progressive roadmap.
type RoadmapHandler struct {
	roadmapSvc *roadmap.Service
}

// NewRoadmapHandler constructs a new RoadmapHandler instance.
func NewRoadmapHandler(roadmapSvc *roadmap.Service) *RoadmapHandler {
	return &RoadmapHandler{
		roadmapSvc: roadmapSvc,
	}
}

// GetCurrentRoadmap returns the dynamic roadmap for the authenticated user,
// presenting ONLY the current active section with all its tutorials and questions.
// GET /api/v1/roadmap
func (h *RoadmapHandler) GetCurrentRoadmap(w http.ResponseWriter, r *http.Request) {
	userIDStr, ok := middleware.GetUserID(r.Context())
	if !ok || userIDStr == "" {
		userIDStr = "00000000-0000-0000-0000-000000000001"
	}

	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		// Fallback for mock IDs
		userUUID = uuid.NewMD5(uuid.NameSpaceOID, []byte(userIDStr))
	}

	res, err := h.roadmapSvc.GetCurrentRoadmap(r.Context(), userUUID)
	if err != nil {
		slog.Error("failed to get current roadmap", "user_id", userIDStr, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get roadmap: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, res)
}

// CompleteNode marks a roadmap node/tutorial as mastered.
// POST /api/v1/roadmap/nodes/{id}/complete
func (h *RoadmapHandler) CompleteNode(w http.ResponseWriter, r *http.Request) {
	userIDStr, ok := middleware.GetUserID(r.Context())
	if !ok || userIDStr == "" {
		userIDStr = "00000000-0000-0000-0000-000000000001"
	}

	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		userUUID = uuid.NewMD5(uuid.NameSpaceOID, []byte(userIDStr))
	}

	nodeIDStr := chi.URLParam(r, "id")
	nodeUUID, err := uuid.Parse(nodeIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid node id parameter")
		return
	}

	if err := h.roadmapSvc.CompleteNode(r.Context(), userUUID, nodeUUID); err != nil {
		slog.Error("failed to complete roadmap node", "user_id", userIDStr, "node_id", nodeIDStr, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to complete node: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "node marked as completed",
	})
}

// TestOutNode tests out of a node to bypass prerequisites.
// POST /api/v1/roadmap/nodes/{id}/test-out
func (h *RoadmapHandler) TestOutNode(w http.ResponseWriter, r *http.Request) {
	userIDStr, ok := middleware.GetUserID(r.Context())
	if !ok || userIDStr == "" {
		userIDStr = "00000000-0000-0000-0000-000000000001"
	}

	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		userUUID = uuid.NewMD5(uuid.NameSpaceOID, []byte(userIDStr))
	}

	nodeIDStr := chi.URLParam(r, "id")
	nodeUUID, err := uuid.Parse(nodeIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid node id parameter")
		return
	}

	if err := h.roadmapSvc.TestOut(r.Context(), userUUID, nodeUUID); err != nil {
		slog.Error("failed to test out of roadmap node", "user_id", userIDStr, "node_id", nodeIDStr, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to test out: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "node successfully tested out",
	})
}

type generateRoadmapPayload struct {
	TargetRole          string   `json:"target_role"`
	ConfirmedHypotheses []string `json:"confirmed_hypotheses,omitempty"`
	DebunkedHypotheses  []string `json:"debunked_hypotheses,omitempty"`
}

// GenerateRoadmap initializes or re-generates a personalized roadmap for the user.
// POST /api/v1/roadmap/generate
func (h *RoadmapHandler) GenerateRoadmap(w http.ResponseWriter, r *http.Request) {
	userIDStr, ok := middleware.GetUserID(r.Context())
	if !ok || userIDStr == "" {
		userIDStr = "00000000-0000-0000-0000-000000000001"
	}

	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		userUUID = uuid.NewMD5(uuid.NameSpaceOID, []byte(userIDStr))
	}

	var payload generateRoadmapPayload
	if r.ContentLength > 0 {
		_ = decodeJSON(w, r, &payload)
	}
	if payload.TargetRole == "" {
		payload.TargetRole = "Software Engineer - DSA & Problem Solving"
	}

	req := roadmap.GenerateRequest{
		UserID:              userUUID,
		TargetRole:          payload.TargetRole,
		ConfirmedHypotheses: payload.ConfirmedHypotheses,
		DebunkedHypotheses:  payload.DebunkedHypotheses,
	}

	if err := h.roadmapSvc.Generate(r.Context(), req); err != nil {
		slog.Error("failed to generate roadmap", "user_id", userIDStr, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to generate roadmap: "+err.Error())
		return
	}

	// Return the newly generated roadmap immediately
	res, err := h.roadmapSvc.GetCurrentRoadmap(r.Context(), userUUID)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "roadmap generated successfully",
		})
		return
	}

	writeJSON(w, http.StatusOK, res)
}
