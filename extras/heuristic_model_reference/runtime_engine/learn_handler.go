package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"velocity/internal/middleware"
	"velocity/internal/models"
	"velocity/internal/services"
)

type LearnHandler struct {
	svc *services.LearnService
}

func NewLearnHandler(svc *services.LearnService) *LearnHandler {
	return &LearnHandler{
		svc: svc,
	}
}

func (h *LearnHandler) Register(r chiRouter) {
	r.Get("/chapters", h.GetChapters)
	r.Get("/page", h.GetLearnPage)
	r.Post("/start", h.Start)
	r.Post("/submit", h.Submit)
	r.Post("/skip", h.Skip)
	r.Post("/close", h.Close)
	r.Get("/session", h.GetSession)
	r.Get("/session/analysis", h.GetAnalysis)
	r.Get("/history", h.GetHistory)
	r.Get("/filters", h.GetFilters)
	r.Post("/similar", h.PickSimilar)
	r.Post("/seek", h.SeekToQuestion) // navigate to any question in ordered scope
}

func (h *LearnHandler) GetChapters(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, "UNAUTHORIZED", "user not authenticated", http.StatusUnauthorized)
		return
	}

	nodes, err := h.svc.GetChapters(r.Context(), userID)
	if err != nil {
		log.Printf("learn/chapters: %v", err)
		writeError(w, "FETCH_FAILED", err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, nodes)
}

func (h *LearnHandler) GetLearnPage(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, "UNAUTHORIZED", "user not authenticated", http.StatusUnauthorized)
		return
	}

	resp, err := h.svc.GetLearnPage(r.Context(), userID)
	if err != nil {
		log.Printf("learn/page: %v", err)
		writeError(w, "FETCH_FAILED", err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *LearnHandler) Start(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, "UNAUTHORIZED", "user not authenticated", http.StatusUnauthorized)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	var req models.StartLearnSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "INVALID_REQUEST", "invalid JSON body", http.StatusBadRequest)
		return
	}

	if req.Mode == "" {
		req.Mode = "ADAPTIVE"
	}

	chapters := req.Chapters
	if req.Chapter != "" {
		chapters = append(chapters, req.Chapter)
	}

	result, err := h.svc.Start(r.Context(), userID, req.Mode, chapters, req.ChapterGroups, req.Subjects, req.Years, req.ExamTypes, req.QuestionLimit, req.BiometricEnabled, req.QuestionOrdering)
	if err != nil {
		log.Printf("learn/start: %v", err)
		writeError(w, "START_FAILED", err.Error(), http.StatusInternalServerError)
		return
	}

	firstQP := models.ToQuestionPayload(result.FirstQuestion)

	resp := models.StartLearnSessionResponse{
		SessionID:           result.Session.ID,
		Mode:                result.Session.Mode,
		Chapter:             result.Session.Chapter,
		ThetaStart:          result.ThetaStart,
		FirstQuestion:       firstQP,
		QuestionMap:         result.QuestionMap,
		QuestionOrdering:    result.Session.QuestionOrdering,
		BiometricEnabled:    result.Session.BiometricEnabled,
		TotalScopeQuestions: result.TotalScopeQuestions,
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *LearnHandler) Submit(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, "UNAUTHORIZED", "user not authenticated", http.StatusUnauthorized)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	var req models.LearnSubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "INVALID_REQUEST", "invalid JSON body", http.StatusBadRequest)
		return
	}

	sessionID := chi.URLParam(r, "sessionID")
	if sessionID == "" {
		sessionID = r.URL.Query().Get("session_id")
	}
	if sessionID == "" {
		writeError(w, "MISSING_SESSION", "session_id is required", http.StatusBadRequest)
		return
	}

	result, err := h.svc.Submit(r.Context(), userID, sessionID, req.QuestionID, req.SelectedOptions, req.TimeTakenMs)
	if err != nil {
		log.Printf("learn/submit: %v", err)
		writeError(w, "SUBMIT_FAILED", err.Error(), http.StatusInternalServerError)
		return
	}

	var nextQ *models.QuestionPayload
	if result.NextQuestion != nil {
		qp := models.ToQuestionPayload(result.NextQuestion)
		nextQ = &qp
	}

	resp := models.LearnSubmitResponse{
		IsCorrect:      result.IsCorrect,
		CorrectOptions: result.CorrectOptions,
		ThetaBefore:    result.ThetaBefore,
		ThetaAfter:     result.ThetaAfter,
		NextQuestion:   nextQ,
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *LearnHandler) Skip(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, "UNAUTHORIZED", "user not authenticated", http.StatusUnauthorized)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	var req models.LearnSkipRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "INVALID_REQUEST", "invalid JSON body", http.StatusBadRequest)
		return
	}

	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		writeError(w, "MISSING_SESSION", "session_id is required", http.StatusBadRequest)
		return
	}

	result, err := h.svc.Skip(r.Context(), userID, sessionID, req.QuestionID, req.TimeTakenMs)
	if err != nil {
		log.Printf("learn/skip: %v", err)
		writeError(w, "SKIP_FAILED", err.Error(), http.StatusInternalServerError)
		return
	}

	var nextQ *models.QuestionPayload
	if result.NextQuestion != nil {
		qp := models.ToQuestionPayload(result.NextQuestion)
		nextQ = &qp
	}

	resp := models.LearnSkipResponse{
		Skipped:      true,
		ThetaBefore:  result.ThetaBefore,
		ThetaAfter:   result.ThetaAfter,
		NextQuestion: nextQ,
	}

	writeJSON(w, http.StatusOK, resp)
}

// SeekToQuestion navigates the session to a specific question from the ordered scope.
func (h *LearnHandler) SeekToQuestion(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, "UNAUTHORIZED", "user not authenticated", http.StatusUnauthorized)
		return
	}

	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		writeError(w, "MISSING_SESSION", "session_id is required", http.StatusBadRequest)
		return
	}

	questionID := r.URL.Query().Get("question_id")
	if questionID == "" {
		writeError(w, "MISSING_QUESTION", "question_id is required", http.StatusBadRequest)
		return
	}

	q, err := h.svc.SeekToQuestion(r.Context(), userID, sessionID, questionID)
	if err != nil {
		log.Printf("learn/seek: %v", err)
		writeError(w, "SEEK_FAILED", err.Error(), http.StatusInternalServerError)
		return
	}

	qp := models.ToQuestionPayload(q)
	writeJSON(w, http.StatusOK, qp)
}

func (h *LearnHandler) Close(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, "UNAUTHORIZED", "user not authenticated", http.StatusUnauthorized)
		return
	}

	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		writeError(w, "MISSING_SESSION", "session_id is required", http.StatusBadRequest)
		return
	}

	result, err := h.svc.Close(r.Context(), userID, sessionID)
	if err != nil {
		log.Printf("learn/close: %v", err)
		writeError(w, "CLOSE_FAILED", err.Error(), http.StatusInternalServerError)
		return
	}

	resp := models.CloseLearnSessionResponse{
		SessionID:      result.SessionID,
		Chapter:        result.Chapter,
		ThetaStart:     result.ThetaStart,
		ThetaFinal:     result.ThetaFinal,
		MasteryScore:   result.MasteryScore,
		TotalQuestions: result.TotalQuestions,
		CorrectCount:   result.CorrectCount,
		Accuracy:       result.Accuracy,
		AvgTimeTakenMs: result.AvgTimeTakenMs,
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *LearnHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, "UNAUTHORIZED", "user not authenticated", http.StatusUnauthorized)
		return
	}

	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		writeError(w, "MISSING_SESSION", "session_id is required", http.StatusBadRequest)
		return
	}

	resp, err := h.svc.GetSession(r.Context(), userID, sessionID)
	if err != nil {
		log.Printf("learn/session: %v", err)
		writeError(w, "FETCH_FAILED", err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *LearnHandler) GetAnalysis(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, "UNAUTHORIZED", "user not authenticated", http.StatusUnauthorized)
		return
	}

	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		writeError(w, "MISSING_SESSION", "session_id is required", http.StatusBadRequest)
		return
	}

	resp, err := h.svc.GetSessionAnalysis(r.Context(), userID, sessionID)
	if err != nil {
		log.Printf("learn/analysis: %v", err)
		writeError(w, "FETCH_FAILED", err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *LearnHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, "UNAUTHORIZED", "user not authenticated", http.StatusUnauthorized)
		return
	}

	resp, err := h.svc.GetSessionHistory(r.Context(), userID)
	if err != nil {
		log.Printf("learn/history: %v", err)
		writeError(w, "FETCH_FAILED", err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *LearnHandler) GetFilters(w http.ResponseWriter, r *http.Request) {
	result, err := h.svc.GetFilters(r.Context())
	if err != nil {
		log.Printf("learn/filters: %v", err)
		writeError(w, "FETCH_FAILED", err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *LearnHandler) PickSimilar(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, "UNAUTHORIZED", "user not authenticated", http.StatusUnauthorized)
		return
	}

	sessionID := r.URL.Query().Get("session_id")
	questionID := r.URL.Query().Get("question_id")

	if sessionID == "" && questionID == "" {
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
		var req struct {
			SessionID  string `json:"session_id"`
			QuestionID string `json:"question_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			sessionID = req.SessionID
			questionID = req.QuestionID
		}
	}

	if sessionID == "" || questionID == "" {
		writeError(w, "MISSING_PARAMS", "session_id and question_id are required", http.StatusBadRequest)
		return
	}

	q, err := h.svc.PickSimilarQuestion(r.Context(), userID, sessionID, questionID)
	if err != nil {
		log.Printf("learn/pick-similar: %v", err)
		writeError(w, "PICK_FAILED", err.Error(), http.StatusInternalServerError)
		return
	}

	qp := models.ToQuestionPayload(q)

	countStr := r.URL.Query().Get("count")
	count := 1
	if c, err := strconv.Atoi(countStr); err == nil && c > 0 {
		count = c
	}

	if count <= 1 {
		writeJSON(w, http.StatusOK, qp)
		return
	}

	var questions []models.QuestionPayload
	for i := range count {
		if i > 0 {
			q, err = h.svc.PickSimilarQuestion(r.Context(), userID, sessionID, questionID)
			if err != nil {
				break
			}
		}
		qp := models.ToQuestionPayload(q)
		questions = append(questions, qp)
	}
	writeJSON(w, http.StatusOK, questions)
}
