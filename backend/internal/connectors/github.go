package connectors

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"transverse/internal/config"
)

type GithubConnector struct {
	cfg        *config.Config
	httpClient *http.Client
}

func NewGithubConnector(cfg *config.Config) *GithubConnector {
	return &GithubConnector{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.ConnectorTimeoutSeconds) * time.Second,
		},
	}
}

type ghRepo struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Topics      []string `json:"topics"`
	Language    string   `json:"language"`
	Owner       struct {
		Login string `json:"login"`
	} `json:"owner"`
}

func (c *GithubConnector) Fetch(ctx context.Context, username string) (*RawSignal, error) {
	reposURL := fmt.Sprintf("%s/users/%s/repos?per_page=%d&sort=updated", c.cfg.GithubAPIBase, username, c.cfg.ConnectorMaxReposScanned)
	
	req, err := http.NewRequestWithContext(ctx, "GET", reposURL, nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api error: status %d", resp.StatusCode)
	}

	var repos []ghRepo
	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		return nil, err
	}

	raw := &RawSignal{
		Languages:     make(map[string]float64),
		ClaimedTopics: []string{},
		Signals:       []Signal{},
	}

	if len(repos) == 0 {
		return raw, nil
	}
	if len(repos) > c.cfg.ConnectorMaxReposScanned {
		repos = repos[:c.cfg.ConnectorMaxReposScanned]
	}

	langBytes := make(map[string]float64)
	var totalBytes float64

	for _, r := range repos {
		// Collect topics
		raw.ClaimedTopics = append(raw.ClaimedTopics, r.Topics...)
		if r.Description != "" {
			raw.ClaimedTopics = append(raw.ClaimedTopics, r.Name) // Also consider name as potential signal
		}

		// Fetch languages
		langs, err := c.fetchLanguages(ctx, r.Owner.Login, r.Name)
		if err == nil {
			for l, b := range langs {
				langBytes[l] += float64(b)
				totalBytes += float64(b)
			}
		}

		// Fetch README
		readmeText, err := c.fetchReadme(ctx, r.Owner.Login, r.Name)
		if err == nil && len(readmeText) > 0 {
			// Basic keyword check for demonstration (could be expanded)
			keywords := []string{"dijkstra", "bfs", "dfs", "dynamic-programming", "graphs", "tree", "binary-search"}
			lowerReadme := strings.ToLower(readmeText)
			for _, kw := range keywords {
				if strings.Contains(lowerReadme, kw) {
					raw.Signals = append(raw.Signals, Signal{
						TopicTag: kw,
						Evidence: fmt.Sprintf("mentioned %s in repo %s README", kw, r.Name),
						Strength: "weak",
					})
				}
			}
		}
	}

	// Calculate language percentages
	if totalBytes > 0 {
		for l, b := range langBytes {
			raw.Languages[l] = b / totalBytes
		}
	}

	return raw, nil
}

func (c *GithubConnector) fetchLanguages(ctx context.Context, owner, repo string) (map[string]int, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/languages", c.cfg.GithubAPIBase, owner, repo)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	var langs map[string]int
	if err := json.NewDecoder(resp.Body).Decode(&langs); err != nil {
		return nil, err
	}
	return langs, nil
}

func (c *GithubConnector) fetchReadme(ctx context.Context, owner, repo string) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/readme", c.cfg.GithubAPIBase, owner, repo)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	c.setHeaders(req)
	// Ask for raw content
	req.Header.Set("Accept", "application/vnd.github.v3.raw")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}

	// Limit to first N chars
	limitReader := io.LimitReader(resp.Body, 5000)
	body, err := io.ReadAll(limitReader)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func (c *GithubConnector) setHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if c.cfg.GithubToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.GithubToken)
	}
}
