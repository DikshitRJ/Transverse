package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"transverse/internal/config"
	"transverse/internal/models"
)

func TestJudge0Service_ExecuteMultipleTestCases(t *testing.T) {
	submissionsCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/submissions" {
			submissionsCount++
			token := "token-test"
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(judge0Submission{Token: token})
			return
		}

		if r.Method == http.MethodGet && r.URL.Path == "/submissions/token-test" {
			var resp judge0Verdict
			resp.Status.ID = Judge0Accepted
			resp.Status.Description = "Accepted"
			timeStr := "0.015"
			resp.Time = &timeStr
			mem := 1200
			resp.Memory = &mem
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(resp)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cfg := &config.Config{
		Judge0BaseURL:   server.URL,
		Judge0TimeoutMs: 2000,
	}

	j0 := NewJudge0Service(cfg)

	req := models.BatchExecuteRequest{
		LanguageID: 71,
		SourceCode: "print(input())",
		TestCases: []models.TestCase{
			{Input: "hello", Output: "hello"},
			{Input: "world", Output: "world"},
			{Input: "42", Output: "42"},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := j0.ExecuteMultipleTestCases(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error in batch execution: %v", err)
	}

	if !res.AllPassed {
		t.Errorf("expected all passed, got false")
	}
	if res.PassedCount != 3 {
		t.Errorf("expected passed count 3, got %d", res.PassedCount)
	}
	if res.TotalCount != 3 {
		t.Errorf("expected total count 3, got %d", res.TotalCount)
	}
	if res.OverallStatus != "Accepted" {
		t.Errorf("expected overall status 'Accepted', got %q", res.OverallStatus)
	}
	if len(res.TestCases) != 3 {
		t.Errorf("expected 3 test case results, got %d", len(res.TestCases))
	}
	if submissionsCount != 3 {
		t.Errorf("expected 3 submissions to Judge0, got %d", submissionsCount)
	}
}

func TestJudge0Service_CompilationErrorEarlyStop(t *testing.T) {
	submissionsCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/submissions" {
			submissionsCount++
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(judge0Submission{Token: "token-ce"})
			return
		}

		if r.Method == http.MethodGet && r.URL.Path == "/submissions/token-ce" {
			var resp judge0Verdict
			resp.Status.ID = Judge0CompilationError
			resp.Status.Description = "Compilation Error"
			ceOut := "syntax error"
			resp.CompileOutput = &ceOut
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(resp)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cfg := &config.Config{
		Judge0BaseURL:   server.URL,
		Judge0TimeoutMs: 2000,
	}

	j0 := NewJudge0Service(cfg)

	req := models.BatchExecuteRequest{
		LanguageID: 54, // C++
		SourceCode: "invalid cpp code",
		TestCases: []models.TestCase{
			{Input: "1", Output: "1"},
			{Input: "2", Output: "2"},
			{Input: "3", Output: "3"},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := j0.ExecuteMultipleTestCases(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error in batch execution: %v", err)
	}

	if res.AllPassed {
		t.Errorf("expected all passed to be false")
	}
	if res.OverallStatusID != Judge0CompilationError {
		t.Errorf("expected overall status ID %d, got %d", Judge0CompilationError, res.OverallStatusID)
	}
	// Compilation error should break on first test case and not run subsequent test cases
	if submissionsCount != 1 {
		t.Errorf("expected early stop with 1 submission, got %d", submissionsCount)
	}
}
