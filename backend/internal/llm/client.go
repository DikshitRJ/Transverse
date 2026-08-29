package llm

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"transverse/internal/config"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type CompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float32   `json:"temperature,omitempty"`
}

type CompletionResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

type Client interface {
	Complete(ctx context.Context, req CompletionRequest, useCache bool) (string, error)
}

type zaiClient struct {
	cfg    *config.Config
	client *http.Client
	rdb    *redis.Client
}

func NewZaiClient(cfg *config.Config, rdb *redis.Client) Client {
	return &zaiClient{
		cfg: cfg,
		client: &http.Client{
			Timeout: time.Duration(cfg.ZaiTimeoutSecs) * time.Second,
		},
		rdb: rdb,
	}
}

func (c *zaiClient) Complete(ctx context.Context, req CompletionRequest, useCache bool) (string, error) {
	if req.Model == "" {
		req.Model = c.cfg.ZaiModel
	}

	// Calculate cache key if caching is enabled
	var cacheKey string
	if useCache && c.rdb != nil {
		reqBytes, _ := json.Marshal(req)
		hash := sha256.Sum256(reqBytes)
		cacheKey = fmt.Sprintf("llm:cache:%s", hex.EncodeToString(hash[:]))

		cached, err := c.rdb.Get(ctx, cacheKey).Result()
		if err == nil && cached != "" {
			return cached, nil
		}
	}

	var lastErr error
	maxRetries := c.cfg.ZaiMaxRetries
	if maxRetries <= 0 {
		maxRetries = 1
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond) // simple backoff
		}

		reqBody, err := json.Marshal(req)
		if err != nil {
			return "", fmt.Errorf("failed to marshal request: %w", err)
		}

		httpReq, err := http.NewRequestWithContext(ctx, "POST", c.cfg.ZaiBaseURL+"/chat/completions", bytes.NewReader(reqBody))
		if err != nil {
			return "", fmt.Errorf("failed to create request: %w", err)
		}

		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+c.cfg.ZaiAPIKey)

		resp, err := c.client.Do(httpReq)
		if err != nil {
			lastErr = fmt.Errorf("http request failed: %w", err)
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			lastErr = fmt.Errorf("failed to read response: %w", err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("api error (status %d): %s", resp.StatusCode, string(respBody))
			continue
		}

		var compResp CompletionResponse
		if err := json.Unmarshal(respBody, &compResp); err != nil {
			lastErr = fmt.Errorf("failed to parse response: %w", err)
			continue
		}

		if len(compResp.Choices) == 0 {
			lastErr = fmt.Errorf("no choices returned")
			continue
		}

		content := compResp.Choices[0].Message.Content

		if useCache && c.rdb != nil {
			// Cache for 24h
			_ = c.rdb.Set(ctx, cacheKey, content, 24*time.Hour).Err()
		}

		return content, nil
	}

	return "", fmt.Errorf("max retries exceeded, last error: %v", lastErr)
}
