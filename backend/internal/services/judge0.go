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

// Judge0 execution status IDs.
const (
	Judge0InQueue             = 1
	Judge0Processing          = 2
	Judge0Accepted            = 3
	Judge0WrongAnswer         = 4
	Judge0TimeLimitExceeded   = 5
	Judge0CompilationError    = 6
	Judge0RuntimeErrorSIGSEGV  = 7
	Judge0RuntimeErrorSIGXFSZ = 8
	Judge0RuntimeErrorSIGFPE  = 9
	Judge0RuntimeErrorSIGABRT = 10
	Judge0RuntimeErrorNZEC    = 11
	Judge0RuntimeErrorOther   = 12
	Judge0InternalError       = 13
	Judge0ExecFormatError     = 14
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
		timeout = 15 * time.Second
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

// ExecuteSync submits code and polls Judge0 until a final verdict is reached or context expires.
func (j *Judge0Service) ExecuteSync(ctx context.Context, req SubmitCodeRequest) (models.VerdictDetail, string, error) {
	token, err := j.SubmitCode(ctx, req)
	if err != nil {
		return models.VerdictDetail{}, "", err
	}

	ticker := time.NewTicker(150 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return models.VerdictDetail{}, token, ctx.Err()
		case <-ticker.C:
			verdict, err := j.GetVerdict(ctx, token)
			if err != nil {
				return models.VerdictDetail{}, token, err
			}
			// Status > 2 means finished (1: In Queue, 2: Processing, 3+: Finished)
			if verdict.StatusID > 2 {
				return verdict, token, nil
			}
		}
	}
}

// ExecuteMultipleTestCases runs the given source code against each test case individually,
// collecting metrics, verifying output match, and calculating an overall verdict.
func (j *Judge0Service) ExecuteMultipleTestCases(ctx context.Context, req models.BatchExecuteRequest) (*models.BatchExecuteResponse, error) {
	if len(req.TestCases) == 0 {
		return nil, fmt.Errorf("no test cases provided for batch execution")
	}

	var results []models.TestCaseResult
	passedCount := 0
	maxTimeMs := 0
	maxMemoryKB := 0
	overallStatusID := Judge0Accepted
	overallStatusDesc := "Accepted"

	for i, tc := range req.TestCases {
		verdict, _, err := j.ExecuteSync(ctx, SubmitCodeRequest{
			SourceCode:     req.SourceCode,
			LanguageID:     req.LanguageID,
			Stdin:          tc.Input,
			ExpectedOutput: tc.Output,
		})
		if err != nil {
			return nil, fmt.Errorf("failed executing testcase %d: %w", i+1, err)
		}

		if verdict.TimeMs > maxTimeMs {
			maxTimeMs = verdict.TimeMs
		}
		if verdict.MemoryKB > maxMemoryKB {
			maxMemoryKB = verdict.MemoryKB
		}

		passed := (verdict.StatusID == Judge0Accepted)
		if passed {
			passedCount++
		} else {
			// If not accepted, update overall status priority:
			// Compilation Error > Runtime Error > TLE > Wrong Answer
			if verdict.StatusID == Judge0CompilationError || overallStatusID == Judge0Accepted {
				overallStatusID = verdict.StatusID
				overallStatusDesc = verdict.StatusDesc
			}
		}

		results = append(results, models.TestCaseResult{
			Index:          i + 1,
			Input:          tc.Input,
			ExpectedOutput: tc.Output,
			Stderr:         verdict.Stderr,
			CompileOutput:  verdict.CompileOutput,
			StatusID:       verdict.StatusID,
			StatusDesc:     verdict.StatusDesc,
			TimeMs:         verdict.TimeMs,
			MemoryKB:       verdict.MemoryKB,
			Passed:         passed,
		})

		// If compilation error occurs, subsequent test cases will also fail compilation; break early
		if verdict.StatusID == Judge0CompilationError {
			break
		}
	}

	allPassed := (passedCount == len(req.TestCases))
	if !allPassed && overallStatusID == Judge0Accepted {
		overallStatusID = Judge0WrongAnswer
		overallStatusDesc = "Wrong Answer"
	}

	return &models.BatchExecuteResponse{
		AllPassed:       allPassed,
		PassedCount:     passedCount,
		TotalCount:      len(req.TestCases),
		OverallStatus:   overallStatusDesc,
		OverallStatusID: overallStatusID,
		MaxTimeMs:       maxTimeMs,
		MaxMemoryKB:     maxMemoryKB,
		TestCases:       results,
	}, nil
}

// IsAccepted returns true for Judge0 status 3 (Accepted).
func IsAccepted(statusID int) bool {
	return statusID == Judge0Accepted
}

// IsCompileError returns true for Judge0 status 6.
func IsCompileError(statusID int) bool {
	return statusID == Judge0CompilationError
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
