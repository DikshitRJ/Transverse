package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"transverse/internal/evidence"
	"transverse/internal/middleware"
	"transverse/internal/models"
)

type EvidenceHandler struct {
	svc *evidence.Service
}

func NewEvidenceHandler(svc *evidence.Service) *EvidenceHandler {
	return &EvidenceHandler{svc: svc}
}

type uploadURLRequest struct {
	Kind     models.EvidenceKind `json:"kind"`
	Filename string              `json:"filename"`
}

type connectorRequest struct {
	Username string `json:"username"`
	Handle   string `json:"handle"`
}

func (h *EvidenceHandler) HandleUploadURL(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "User ID not found in context")
		return
	}

	var req uploadURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Kind != models.EvidenceKindResume && req.Kind != models.EvidenceKindCodebase {
		writeError(w, http.StatusBadRequest, "invalid kind for upload")
		return
	}

	srcID, url, err := h.svc.GeneratePresignedUpload(r.Context(), userID, req.Kind, req.Filename)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"evidence_id": srcID,
		"upload_url":  url,
	})
}

func (h *EvidenceHandler) HandleConfirmUpload(w http.ResponseWriter, r *http.Request) {
	evidenceID := chi.URLParam(r, "id")
	if err := h.svc.ConfirmUpload(r.Context(), evidenceID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "processing_queued"})
}

func (h *EvidenceHandler) HandleGithub(w http.ResponseWriter, r *http.Request) {
	h.handleConnector(w, r, models.EvidenceKindGithub, "username")
}

func (h *EvidenceHandler) HandleLeetcode(w http.ResponseWriter, r *http.Request) {
	h.handleConnector(w, r, models.EvidenceKindLeetcode, "username")
}

func (h *EvidenceHandler) HandleCodeforces(w http.ResponseWriter, r *http.Request) {
	h.handleConnector(w, r, models.EvidenceKindCodeforces, "handle")
}

func (h *EvidenceHandler) handleConnector(w http.ResponseWriter, r *http.Request, kind models.EvidenceKind, keyField string) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "User ID not found in context")
		return
	}

	var req connectorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ref := req.Username
	if keyField == "handle" {
		ref = req.Handle
	}

	srcID, err := h.svc.StartConnectorSource(r.Context(), userID, kind, ref)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{
		"evidence_id": srcID,
		"status":      "processing_queued",
	})
}
