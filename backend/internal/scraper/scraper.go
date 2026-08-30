package scraper

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"transverse/internal/models"
	"transverse/internal/templates"
)

// ScrapedProblem contains all extracted information from a problem page.
type ScrapedProblem struct {
	Title       string            `json:"title"`
	Source      string            `json:"source"`
	URL         string            `json:"url"`
	Statement   string            `json:"statement"`
	TimeLimit   string            `json:"time_limit,omitempty"`
	MemoryLimit string            `json:"memory_limit,omitempty"`
	InputSpec   string            `json:"input_specification,omitempty"`
	OutputSpec  string            `json:"output_specification,omitempty"`
	TestCases   []models.TestCase `json:"test_cases"`
	Tags        []string          `json:"tags,omitempty"`
	Difficulty  string            `json:"difficulty,omitempty"`
	Templates   map[string]string `json:"templates,omitempty"`
}

// ProblemScraper defines the interface for scraping problem data from CP platforms.
type ProblemScraper interface {
	Scrape(ctx context.Context, problemURL string) (*ScrapedProblem, error)
}

// UnifiedScraper routes URLs to platform-specific scrapers.
type UnifiedScraper struct {
	codeforces *CodeforcesScraper
	leetcode   *LeetCodeScraper
	generic    *GenericScraper
}

// NewUnifiedScraper creates a new unified problem scraper.
func NewUnifiedScraper(timeout time.Duration) *UnifiedScraper {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &UnifiedScraper{
		codeforces: NewCodeforcesScraper(timeout),
		leetcode:   NewLeetCodeScraper(timeout),
		generic:    NewGenericScraper(timeout),
	}
}

// Scrape routes the problem URL to the appropriate scraper and enriches it with starter code templates.
func (u *UnifiedScraper) Scrape(ctx context.Context, rawURL string) (*ScrapedProblem, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, fmt.Errorf("invalid problem url: %w", err)
	}

	host := strings.ToLower(parsed.Host)
	var scraped *ScrapedProblem

	switch {
	case strings.Contains(host, "codeforces.com"):
		scraped, err = u.codeforces.Scrape(ctx, rawURL)
	case strings.Contains(host, "leetcode.com"):
		scraped, err = u.leetcode.Scrape(ctx, rawURL)
	default:
		scraped, err = u.generic.Scrape(ctx, rawURL)
	}

	if err != nil {
		return nil, err
	}

	// Enrich with generated template code if not provided by source
	if len(scraped.Templates) == 0 {
		slug := ""
		if scraped.URL != "" {
			parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
			if len(parts) > 0 {
				slug = parts[len(parts)-1]
			}
		}
		scraped.Templates = templates.GenerateTemplates(scraped.Title, slug, "")
	}

	return scraped, nil
}
