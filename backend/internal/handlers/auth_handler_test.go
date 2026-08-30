package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"transverse/internal/cache"
	"transverse/internal/config"
)

// TestAuthHandler_Validation tests validation logic on Register and Login requests
func TestAuthHandler_Validation(t *testing.T) {
	cfg := &config.Config{
		JWTSecret:           "test-secret-1234567890-test-secret",
		JWTAccessTTLMinutes: 15,
		JWTRefreshTTLDays:   30,
		FrontendOrigin:      "http://localhost:3000",
	}
	appCache := cache.NewMemoryCache()
	// userRepo and oauthRepo nil for validation checks that reject early
	handler := NewAuthHandler(cfg, nil, nil, appCache)

	t.Run("Register missing email", func(t *testing.T) {
		body, _ := json.Marshal(RegisterRequest{
			Email:    "",
			Password: "password123",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
		w := httptest.NewRecorder()
		handler.Register(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("Register short password", func(t *testing.T) {
		body, _ := json.Marshal(RegisterRequest{
			Email:    "test@example.com",
			Password: "123",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
		w := httptest.NewRecorder()
		handler.Register(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("Login missing credentials", func(t *testing.T) {
		body, _ := json.Marshal(LoginRequest{
			Email:    "",
			Password: "",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
		w := httptest.NewRecorder()
		handler.Login(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("Unsupported OAuth Provider Google rejected", func(t *testing.T) {
		_, err := handler.getConfig("google")
		if err == nil {
			t.Fatalf("expected error for google provider after removal, got nil")
		}
	})

	t.Run("Supported OAuth Provider Github accepted", func(t *testing.T) {
		conf, err := handler.getConfig("github")
		if err != nil || conf == nil {
			t.Fatalf("expected github provider to be supported, got err: %v", err)
		}
	})
}
