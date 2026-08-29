// Package services provides business logic for the practice engine, scoring, and Judge0 proxy integration.
package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"transverse/internal/config"
	"transverse/internal/models"
)

// Judge0Service handles all communication with the Judge0 API.
type Judge0Service struct {
	client  *http.Client
	baseURL string
	apiKey  string
	apiHost string
}

// NewJudge0Service creates a new Judge0Service instance configured from application settings.
func NewJudge0Service(cfg *config.Config) *Judge0Service {
	timeout := time.Duration(cfg.Judge0TimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	baseURL := strings.TrimRight(cfg.Judge0BaseURL, "/")

	var apiHost string
	if parsedURL, err := url.Parse(baseURL); err == nil {
		apiHost = parsedURL.Host
	}

	return &Judge0Service{
		client: &http.Client{
			Timeout: timeout,
		},
		baseURL: baseURL,
		apiKey:  cfg.Judge0APIKey,
		apiHost: apiHost,
	}
}

// SubmitCodeRequest specifies the parameters for submitting source code for execution.
type SubmitCodeRequest struct {
	SourceCode     string `json:"source_code"`
	LanguageID     int    `json:"language_id"`
	Stdin          string `json:"stdin,omitempty"`
	ExpectedOutput string `json:"expected_output,omitempty"`
}

// judge0Submission represents the initial asynchronous submission response from Judge0.
type judge0Submission struct {
	Token string `json:"token"`
}

// judge0Verdict represents the polling response from Judge0 with execution details.
type judge0Verdict struct {
	Status struct {
		ID          int    `json:"id"`
		Description string `json:"description"`
	} `json:"status"`
	Time          *string `json:"time"`
	Memory        *int    `json:"memory"`
	Stdout        *string `json:"stdout"`
	Stderr        *string `json:"stderr"`
	CompileOutput *string `json:"compile_output"`
	Message       *string `json:"message"`
}

// SubmitCode submits source code to Judge0 and returns the submission token asynchronously.
// Does NOT wait for execution completion (verdict polling is done separately).
func (j *Judge0Service) SubmitCode(ctx context.Context, req SubmitCodeRequest) (string, error) {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("judge0: marshal submit request: %w", err)
	}

	endpoint := fmt.Sprintf("%s/submissions?base64_encoded=false&wait=false", j.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("judge0: build submit http request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if j.apiKey != "" {
		httpReq.Header.Set("X-RapidAPI-Key", j.apiKey)
		if j.apiHost != "" {
			httpReq.Header.Set("X-RapidAPI-Host", j.apiHost)
		}
		httpReq.Header.Set("X-Auth-Token", j.apiKey)
	}

	resp, err := j.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("judge0: execute submit request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("judge0: submit returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var sub judge0Submission
	if err := json.NewDecoder(resp.Body).Decode(&sub); err != nil {
		return "", fmt.Errorf("judge0: decode submit response: %w", err)
	}

	if sub.Token == "" {
		return "", fmt.Errorf("judge0: received empty submission token")
	}

	return sub.Token, nil
}

// GetVerdict fetches the execution status and verdict metrics for a given submission token.
func (j *Judge0Service) GetVerdict(ctx context.Context, token string) (models.VerdictDetail, error) {
	endpoint := fmt.Sprintf("%s/submissions/%s?base64_encoded=false&fields=status,time,memory,stdout,stderr,compile_output,message", j.baseURL, url.PathEscape(token))

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return models.VerdictDetail{}, fmt.Errorf("judge0: build verdict http request: %w", err)
	}

	if j.apiKey != "" {
		httpReq.Header.Set("X-RapidAPI-Key", j.apiKey)
		if j.apiHost != "" {
			httpReq.Header.Set("X-RapidAPI-Host", j.apiHost)
		}
		httpReq.Header.Set("X-Auth-Token", j.apiKey)
	}

	resp, err := j.client.Do(httpReq)
	if err != nil {
		return models.VerdictDetail{}, fmt.Errorf("judge0: execute verdict request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return models.VerdictDetail{}, fmt.Errorf("judge0: verdict returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var v judge0Verdict
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return models.VerdictDetail{}, fmt.Errorf("judge0: decode verdict response: %w", err)
	}

	timeMs := 0
	if v.Time != nil && *v.Time != "" {
		if sec, err := strconv.ParseFloat(*v.Time, 64); err == nil {
			timeMs = int(sec * 1000)
		}
	}

	memoryKB := 0
	if v.Memory != nil {
		memoryKB = *v.Memory
	}

	var stderr string
	if v.Stderr != nil {
		stderr = *v.Stderr
	}

	var compileOutput string
	if v.CompileOutput != nil {
		compileOutput = *v.CompileOutput
	}

	var message string
	if v.Message != nil {
		message = *v.Message
	}

	return models.VerdictDetail{
		StatusID:      v.Status.ID,
		StatusDesc:    v.Status.Description,
		TimeMs:        timeMs,
		MemoryKB:      memoryKB,
		Stderr:        stderr,
		CompileOutput: compileOutput,
		Message:       message,
	}, nil
}

// IsAccepted returns true for Judge0 status 3 (Accepted).
func IsAccepted(statusID int) bool {
	return statusID == 3
}

// IsCompileError returns true for Judge0 status 6 or 7.
func IsCompileError(statusID int) bool {
	return statusID == 6 || statusID == 7
}

// SupportedLanguages maps language identifier keys to Judge0 language_ids.
var SupportedLanguages = map[string]int{
	"c":    50,
	"cpp":  54,
	"java": 62,
	"py":   71,
	"js":   63,
	"go":   60,
	"rust": 73,
	"kt":   78,
}
