package scraper

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"transverse/internal/models"
)

// CodeforcesScraper scrapes problem statements and test cases from Codeforces.
type CodeforcesScraper struct {
	httpClient *http.Client
}

// NewCodeforcesScraper creates a new Codeforces scraper instance.
func NewCodeforcesScraper(timeout time.Duration) *CodeforcesScraper {
	return &CodeforcesScraper{
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

var (
	titleRegex       = regexp.MustCompile(`(?s)<div class="title">\s*([^<]+)\s*</div>`)
	timeLimitRegex   = regexp.MustCompile(`(?s)<div class="time-limit">\s*<div class="property-title">[^<]*</div>\s*([^<]+)\s*</div>`)
	memoryLimitRegex = regexp.MustCompile(`(?s)<div class="memory-limit">\s*<div class="property-title">[^<]*</div>\s*([^<]+)\s*</div>`)
	inputSpecRegex   = regexp.MustCompile(`(?s)<div class="input-specification">\s*<div class="section-title">[^<]*</div>\s*(.*?)\s*</div>\s*<div class="output-specification">`)
	outputSpecRegex  = regexp.MustCompile(`(?s)<div class="output-specification">\s*<div class="section-title">[^<]*</div>\s*(.*?)\s*</div>\s*<div class="sample-tests"`)
	sampleInputRegex = regexp.MustCompile(`(?s)<div class="input">\s*<div class="title">[^<]*</div>\s*<pre>(.*?)</pre>`)
	sampleOutputRegex = regexp.MustCompile(`(?s)<div class="output">\s*<div class="title">[^<]*</div>\s*<pre>(.*?)</pre>`)
	htmlTagRegex     = regexp.MustCompile(`(?s)<[^>]+>`)
	statementRegex   = regexp.MustCompile(`(?s)<div class="header">.*?</div>\s*<div>\s*(.*?)\s*</div>\s*<div class="input-specification">`)
)

func cleanHTML(s string) string {
	// Replace <br/> and <br> with newlines
	s = regexp.MustCompile(`(?i)<br\s*/?>`).ReplaceAllString(s, "\n")
	// Replace </p> and </div> with newlines
	s = regexp.MustCompile(`(?i)</(p|div)>`).ReplaceAllString(s, "\n")
	// Strip remaining HTML tags
	s = htmlTagRegex.ReplaceAllString(s, "")
	// Unescape HTML entities
	s = html.UnescapeString(s)
	// Normalize empty lines
	lines := strings.Split(s, "\n")
	var cleaned []string
	for _, l := range lines {
		trimmed := strings.TrimRight(l, " \t\r")
		cleaned = append(cleaned, trimmed)
	}
	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}

func cleanCodeforcesPre(s string) string {
	// Codeforces sample tests sometimes wrap each line in <div class="test-example-line">...</div>
	s = regexp.MustCompile(`(?i)<div class="test-example-line">`).ReplaceAllString(s, "")
	s = regexp.MustCompile(`(?i)</div>`).ReplaceAllString(s, "\n")
	s = regexp.MustCompile(`(?i)<br\s*/?>`).ReplaceAllString(s, "\n")
	s = htmlTagRegex.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	return strings.TrimSpace(s)
}

// Scrape extracts problem details and test cases from Codeforces problem URLs.
func (c *CodeforcesScraper) Scrape(ctx context.Context, problemURL string) (*ScrapedProblem, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, problemURL, nil)
	if err != nil {
		return nil, fmt.Errorf("codeforces: create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("codeforces: fetch page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("codeforces: page returned status %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("codeforces: read body: %w", err)
	}
	htmlContent := string(bodyBytes)

	// Extract Title
	title := ""
	if matches := titleRegex.FindStringSubmatch(htmlContent); len(matches) > 1 {
		title = strings.TrimSpace(html.UnescapeString(matches[1]))
	}

	// Extract Time Limit & Memory Limit
	timeLimit := ""
	if matches := timeLimitRegex.FindStringSubmatch(htmlContent); len(matches) > 1 {
		timeLimit = strings.TrimSpace(matches[1])
	}
	memoryLimit := ""
	if matches := memoryLimitRegex.FindStringSubmatch(htmlContent); len(matches) > 1 {
		memoryLimit = strings.TrimSpace(matches[1])
	}

	// Extract Statement
	statement := ""
	if matches := statementRegex.FindStringSubmatch(htmlContent); len(matches) > 1 {
		statement = cleanHTML(matches[1])
	}

	// Extract Input Spec
	inputSpec := ""
	if matches := inputSpecRegex.FindStringSubmatch(htmlContent); len(matches) > 1 {
		inputSpec = cleanHTML(matches[1])
	}

	// Extract Output Spec
	outputSpec := ""
	if matches := outputSpecRegex.FindStringSubmatch(htmlContent); len(matches) > 1 {
		outputSpec = cleanHTML(matches[1])
	}

	// Build full formatted statement
	var fullStatement strings.Builder
	if statement != "" {
		fullStatement.WriteString(statement)
	}
	if inputSpec != "" {
		fullStatement.WriteString("\n\n### Input\n")
		fullStatement.WriteString(inputSpec)
	}
	if outputSpec != "" {
		fullStatement.WriteString("\n\n### Output\n")
		fullStatement.WriteString(outputSpec)
	}

	// Extract Sample Test Cases
	inputs := sampleInputRegex.FindAllStringSubmatch(htmlContent, -1)
	outputs := sampleOutputRegex.FindAllStringSubmatch(htmlContent, -1)

	var testCases []models.TestCase
	numCases := len(inputs)
	if len(outputs) < numCases {
		numCases = len(outputs)
	}

	for i := 0; i < numCases; i++ {
		in := cleanCodeforcesPre(inputs[i][1])
		out := cleanCodeforcesPre(outputs[i][1])
		if in != "" || out != "" {
			testCases = append(testCases, models.TestCase{
				Input:       in,
				Output:      out,
				Explanation: fmt.Sprintf("Sample Test Case %d", i+1),
				IsHidden:    false,
			})
		}
	}

	return &ScrapedProblem{
		Title:       title,
		Source:      "codeforces",
		URL:         problemURL,
		Statement:   fullStatement.String(),
		TimeLimit:   timeLimit,
		MemoryLimit: memoryLimit,
		InputSpec:   inputSpec,
		OutputSpec:  outputSpec,
		TestCases:   testCases,
	}, nil
}
