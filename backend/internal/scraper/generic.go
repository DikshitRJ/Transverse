package scraper

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	"transverse/internal/models"
)

// GenericScraper extracts problem statement and sample test cases from standard CP pages.
type GenericScraper struct {
	httpClient *http.Client
}

// NewGenericScraper creates a new generic scraper instance.
func NewGenericScraper(timeout time.Duration) *GenericScraper {
	return &GenericScraper{
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

var (
	genericTitleRegex   = regexp.MustCompile(`(?i)<title>\s*([^<]+)\s*</title>`)
	genericH1Regex      = regexp.MustCompile(`(?i)<h1[^>]*>\s*([^<]+)\s*</h1>`)
	genericExampleRegex = regexp.MustCompile(`(?si)(?:Input|Sample Input)[:\s]*<pre[^>]*>(.*?)</pre>.*?[\s\S]*?(?:Output|Sample Output)[:\s]*<pre[^>]*>(.*?)</pre>`)
)

// Scrape extracts general page text and any structured test case pairs.
func (g *GenericScraper) Scrape(ctx context.Context, problemURL string) (*ScrapedProblem, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, problemURL, nil)
	if err != nil {
		return nil, fmt.Errorf("generic scraper: create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("generic scraper: fetch url: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("generic scraper: url returned status %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("generic scraper: read response body: %w", err)
	}
	htmlContent := string(bodyBytes)

	// Extract Title
	title := ""
	if h1Match := genericH1Regex.FindStringSubmatch(htmlContent); len(h1Match) > 1 {
		title = cleanHTML(h1Match[1])
	} else if titleMatch := genericTitleRegex.FindStringSubmatch(htmlContent); len(titleMatch) > 1 {
		title = cleanHTML(titleMatch[1])
	}

	// Extract sample test cases
	var testCases []models.TestCase
	matches := genericExampleRegex.FindAllStringSubmatch(htmlContent, -1)
	for i, m := range matches {
		if len(m) > 2 {
			in := cleanHTML(m[1])
			out := cleanHTML(m[2])
			if in != "" || out != "" {
				testCases = append(testCases, models.TestCase{
					Input:       in,
					Output:      out,
					Explanation: fmt.Sprintf("Sample %d", i+1),
					IsHidden:    false,
				})
			}
		}
	}

	statement := cleanHTML(htmlContent)
	if len(statement) > 3000 {
		statement = statement[:3000] + "..."
	}

	return &ScrapedProblem{
		Title:     title,
		Source:    "generic",
		URL:       problemURL,
		Statement: statement,
		TestCases: testCases,
	}, nil
}
