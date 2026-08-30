package scraper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"transverse/internal/models"
)

// LeetCodeScraper extracts problem content, examples, and starter snippets from LeetCode GraphQL API.
type LeetCodeScraper struct {
	httpClient *http.Client
	graphqlURL string
}

// NewLeetCodeScraper creates a new LeetCode scraper instance.
func NewLeetCodeScraper(timeout time.Duration) *LeetCodeScraper {
	return &LeetCodeScraper{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		graphqlURL: "https://leetcode.com/graphql",
	}
}

type leetcodeQuestionData struct {
	Data struct {
		Question struct {
			QuestionID          string `json:"questionId"`
			Title               string `json:"title"`
			TitleSlug           string `json:"titleSlug"`
			Content             string `json:"content"`
			Difficulty          string `json:"difficulty"`
			ExampleTestcaseList []string `json:"exampleTestcaseList"`
			SampleTestCase      string   `json:"sampleTestCase"`
			CodeSnippets        []struct {
				Lang     string `json:"lang"`
				LangSlug string `json:"langSlug"`
				Code     string `json:"code"`
			} `json:"codeSnippets"`
			TopicTags []struct {
				Name string `json:"name"`
				Slug string `json:"slug"`
			} `json:"topicTags"`
		} `json:"question"`
	} `json:"data"`
}

var (
	lcExampleInputRegex  = regexp.MustCompile(`(?i)(?:<strong>\s*)?Input:?(?:</strong>)?\s*([^\n<]+)`)
	lcExampleOutputRegex = regexp.MustCompile(`(?i)(?:<strong>\s*)?Output:?(?:</strong>)?\s*([^\n<]+)`)
	lcExampleBlockRegex  = regexp.MustCompile(`(?si)<pre>(.*?)</pre>`)
)

func extractLeetcodeSlug(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	for i, p := range parts {
		if p == "problems" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// Scrape queries the LeetCode GraphQL API for detailed problem statement, example test cases, and code snippets.
func (s *LeetCodeScraper) Scrape(ctx context.Context, problemURL string) (*ScrapedProblem, error) {
	slug := extractLeetcodeSlug(problemURL)
	if slug == "" {
		return nil, fmt.Errorf("leetcode: cannot extract problem slug from url %q", problemURL)
	}

	query := `
	query getQuestionDetail($titleSlug: String!) {
		question(titleSlug: $titleSlug) {
			questionId
			title
			titleSlug
			content
			difficulty
			exampleTestcaseList
			sampleTestCase
			codeSnippets {
				lang
				langSlug
				code
			}
			topicTags {
				name
				slug
			}
		}
	}
	`

	reqPayload := map[string]interface{}{
		"query": query,
		"variables": map[string]interface{}{
			"titleSlug": slug,
		},
	}
	payloadBytes, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("leetcode: marshal graphql payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.graphqlURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("leetcode: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("leetcode: execute graphql query: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("leetcode: graphql error status %d: %s", resp.StatusCode, string(body))
	}

	var data leetcodeQuestionData
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("leetcode: decode graphql response: %w", err)
	}

	q := data.Data.Question
	if q.Title == "" {
		return nil, fmt.Errorf("leetcode: problem %q not found", slug)
	}

	// Clean statement
	statementText := cleanHTML(q.Content)

	// Extract tags
	var tags []string
	for _, t := range q.TopicTags {
		tags = append(tags, t.Name)
	}

	// Extract test cases from <pre> blocks in HTML content
	var testCases []models.TestCase
	preBlocks := lcExampleBlockRegex.FindAllStringSubmatch(q.Content, -1)
	for i, block := range preBlocks {
		if len(block) > 1 {
			blockContent := block[1]
			inMatch := lcExampleInputRegex.FindStringSubmatch(blockContent)
			outMatch := lcExampleOutputRegex.FindStringSubmatch(blockContent)

			if len(inMatch) > 1 && len(outMatch) > 1 {
				testCases = append(testCases, models.TestCase{
					Input:       cleanHTML(inMatch[1]),
					Output:      cleanHTML(outMatch[1]),
					Explanation: fmt.Sprintf("Example %d", i+1),
					IsHidden:    false,
				})
			}
		}
	}

	// Fallback to exampleTestcaseList if pre blocks didn't yield structured pairs
	if len(testCases) == 0 && len(q.ExampleTestcaseList) > 0 {
		for i, tc := range q.ExampleTestcaseList {
			testCases = append(testCases, models.TestCase{
				Input:       tc,
				Output:      "",
				Explanation: fmt.Sprintf("Example %d", i+1),
				IsHidden:    false,
			})
		}
	}

	// Extract language templates from codeSnippets
	templateMap := make(map[string]string)
	langMap := map[string]string{
		"python3":    "py",
		"python":     "py",
		"cpp":        "cpp",
		"java":       "java",
		"javascript": "js",
		"golang":     "go",
		"rust":       "rust",
		"c":          "c",
		"kotlin":     "kt",
	}

	for _, snippet := range q.CodeSnippets {
		if mappedKey, ok := langMap[snippet.LangSlug]; ok {
			templateMap[mappedKey] = snippet.Code
		}
	}

	return &ScrapedProblem{
		Title:      q.Title,
		Source:     "leetcode",
		URL:        problemURL,
		Statement:  statementText,
		Difficulty: strings.ToLower(q.Difficulty),
		Tags:       tags,
		TestCases:  testCases,
		Templates:  templateMap,
	}, nil
}
