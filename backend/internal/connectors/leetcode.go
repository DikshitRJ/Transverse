package connectors

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"transverse/internal/config"
)

type LeetcodeConnector struct {
	cfg        *config.Config
	httpClient *http.Client
}

func NewLeetcodeConnector(cfg *config.Config) *LeetcodeConnector {
	return &LeetcodeConnector{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.ConnectorTimeoutSeconds) * time.Second,
		},
	}
}

type leetcodeResponse struct {
	Data struct {
		MatchedUser struct {
			Username            string `json:"username"`
			SubmitStatsGlobal struct {
				AcSubmissionNum []struct {
					Difficulty string `json:"difficulty"`
					Count      int    `json:"count"`
				} `json:"acSubmissionNum"`
			} `json:"submitStatsGlobal"`
			TagProblemCounts struct {
				Advanced []struct {
					TagName        string `json:"tagName"`
					ProblemsSolved int    `json:"problemsSolved"`
				} `json:"advanced"`
				Medium []struct {
					TagName        string `json:"tagName"`
					ProblemsSolved int    `json:"problemsSolved"`
				} `json:"medium"`
				Fundamental []struct {
					TagName        string `json:"tagName"`
					ProblemsSolved int    `json:"problemsSolved"`
				} `json:"fundamental"`
			} `json:"tagProblemCounts"`
		} `json:"matchedUser"`
	} `json:"data"`
}

func (c *LeetcodeConnector) Fetch(ctx context.Context, username string) (*RawSignal, error) {
	query := `
	query getUserProfile($username: String!) {
		matchedUser(username: $username) {
			username
			submitStatsGlobal {
				acSubmissionNum {
					difficulty
					count
				}
			}
			tagProblemCounts {
				advanced {
					tagName
					problemsSolved
				}
				medium {
					tagName
					problemsSolved
				}
				fundamental {
					tagName
					problemsSolved
				}
			}
		}
	}
	`
	body, _ := json.Marshal(map[string]interface{}{
		"query": query,
		"variables": map[string]interface{}{
			"username": username,
		},
	})

	req, err := http.NewRequestWithContext(ctx, "POST", c.cfg.LeetcodeGraphQLURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("leetcode api error: status %d", resp.StatusCode)
	}

	var lcResp leetcodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&lcResp); err != nil {
		return nil, err
	}

	if lcResp.Data.MatchedUser.Username == "" {
		return nil, fmt.Errorf("user not found or private profile")
	}

	raw := &RawSignal{
		Languages:     make(map[string]float64), // Leetcode API doesn't easily expose this in public matchedUser without more complex queries, omit for now.
		ClaimedTopics: []string{},
		Signals:       []Signal{},
	}

	// Add difficulties
	for _, diff := range lcResp.Data.MatchedUser.SubmitStatsGlobal.AcSubmissionNum {
		if diff.Count > 0 {
			raw.Signals = append(raw.Signals, Signal{
				TopicTag: "leetcode",
				Evidence: fmt.Sprintf("Solved %d %s problems on LeetCode", diff.Count, diff.Difficulty),
				Strength: "moderate",
			})
		}
	}

	// Add tags
	addTags := func(tags []struct{TagName string `json:"tagName"`; ProblemsSolved int `json:"problemsSolved"`}) {
		for _, tag := range tags {
			if tag.ProblemsSolved > 0 {
				raw.ClaimedTopics = append(raw.ClaimedTopics, tag.TagName)
				raw.Signals = append(raw.Signals, Signal{
					TopicTag: tag.TagName,
					Evidence: fmt.Sprintf("Solved %d problems under %s tag on LeetCode", tag.ProblemsSolved, tag.TagName),
					Strength: "moderate", // "strong" if high count
				})
			}
		}
	}

	addTags(lcResp.Data.MatchedUser.TagProblemCounts.Advanced)
	addTags(lcResp.Data.MatchedUser.TagProblemCounts.Medium)
	addTags(lcResp.Data.MatchedUser.TagProblemCounts.Fundamental)

	return raw, nil
}
