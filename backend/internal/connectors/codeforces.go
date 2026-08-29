package connectors

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"transverse/internal/config"
)

type CodeforcesConnector struct {
	cfg        *config.Config
	httpClient *http.Client
}

func NewCodeforcesConnector(cfg *config.Config) *CodeforcesConnector {
	return &CodeforcesConnector{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.ConnectorTimeoutSeconds) * time.Second,
		},
	}
}

type cfUserInfoResponse struct {
	Status string `json:"status"`
	Result []struct {
		Handle string `json:"handle"`
		Rating int    `json:"rating"`
		Rank   string `json:"rank"`
	} `json:"result"`
}

type cfUserStatusResponse struct {
	Status string `json:"status"`
	Result []struct {
		Verdict string `json:"verdict"`
		Problem struct {
			Tags []string `json:"tags"`
		} `json:"problem"`
	} `json:"result"`
}

func (c *CodeforcesConnector) Fetch(ctx context.Context, handle string) (*RawSignal, error) {
	// 1. Fetch user info
	infoURL := fmt.Sprintf("%s/user.info?handles=%s", c.cfg.CodeforcesAPIBase, handle)
	reqInfo, err := http.NewRequestWithContext(ctx, "GET", infoURL, nil)
	if err != nil {
		return nil, err
	}

	respInfo, err := c.httpClient.Do(reqInfo)
	if err != nil {
		return nil, err
	}
	defer respInfo.Body.Close()

	if respInfo.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("codeforces info api error: status %d", respInfo.StatusCode)
	}

	var infoResp cfUserInfoResponse
	if err := json.NewDecoder(respInfo.Body).Decode(&infoResp); err != nil {
		return nil, err
	}

	if infoResp.Status != "OK" || len(infoResp.Result) == 0 {
		return nil, fmt.Errorf("codeforces user not found")
	}

	userRating := infoResp.Result[0].Rating
	userRank := infoResp.Result[0].Rank

	// 2. Fetch user status (submissions)
	statusURL := fmt.Sprintf("%s/user.status?handle=%s&from=1&count=500", c.cfg.CodeforcesAPIBase, handle) // Limit to last 500
	reqStatus, err := http.NewRequestWithContext(ctx, "GET", statusURL, nil)
	if err != nil {
		return nil, err
	}

	respStatus, err := c.httpClient.Do(reqStatus)
	if err != nil {
		return nil, err
	}
	defer respStatus.Body.Close()

	if respStatus.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("codeforces status api error: status %d", respStatus.StatusCode)
	}

	var statusResp cfUserStatusResponse
	if err := json.NewDecoder(respStatus.Body).Decode(&statusResp); err != nil {
		return nil, err
	}

	raw := &RawSignal{
		Languages:     make(map[string]float64), // CF status has programmingLanguage, could parse but skip for now
		ClaimedTopics: []string{},
		Signals:       []Signal{},
	}

	if userRating > 0 {
		raw.Signals = append(raw.Signals, Signal{
			TopicTag: "competitive-programming",
			Evidence: fmt.Sprintf("Codeforces rating %d (%s)", userRating, userRank),
			Strength: "strong",
		})
	}

	// Count AC tags
	tagCounts := make(map[string]int)
	for _, sub := range statusResp.Result {
		if sub.Verdict == "OK" {
			for _, tag := range sub.Problem.Tags {
				tagCounts[tag]++
			}
		}
	}

	for tag, count := range tagCounts {
		raw.ClaimedTopics = append(raw.ClaimedTopics, tag)
		raw.Signals = append(raw.Signals, Signal{
			TopicTag: tag,
			Evidence: fmt.Sprintf("Solved %d problems with tag %s on Codeforces", count, tag),
			Strength: "moderate", // Or strong if count is very high
		})
	}

	return raw, nil
}
