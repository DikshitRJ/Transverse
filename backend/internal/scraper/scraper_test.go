package scraper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCodeforcesScraper_ParseSample(t *testing.T) {
	mockHTML := `
	<!DOCTYPE html>
	<html>
	<body>
		<div class="problem-statement">
			<div class="header">
				<div class="title">A. Watermelon</div>
				<div class="time-limit"><div class="property-title">time limit per test</div>1.0s</div>
				<div class="memory-limit"><div class="property-title">memory limit per test</div>64MB</div>
			</div>
			<div><p>One hot summer day Pete and his friend Billy decided to buy a watermelon.</p></div>
			<div class="input-specification">
				<div class="section-title">Input</div>
				<p>The first (and only) input line contains integer number w (1 &le; w &le; 100).</p>
			</div>
			<div class="output-specification">
				<div class="section-title">Output</div>
				<p>Print YES, if the boys can divide the watermelon, and NO otherwise.</p>
			</div>
			<div class="sample-tests">
				<div class="section-title">Examples</div>
				<div class="sample-test">
					<div class="input"><div class="title">Input</div><pre>8</pre></div>
					<div class="output"><div class="title">Output</div><pre>YES</pre></div>
				</div>
			</div>
		</div>
	</body>
	</html>
	`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(mockHTML))
	}))
	defer server.Close()

	scraper := NewCodeforcesScraper(5 * time.Second)
	problem, err := scraper.Scrape(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("unexpected error scraping: %v", err)
	}

	if problem.Title != "A. Watermelon" {
		t.Errorf("expected title 'A. Watermelon', got %q", problem.Title)
	}
	if problem.TimeLimit != "1.0s" {
		t.Errorf("expected time limit '1.0s', got %q", problem.TimeLimit)
	}
	if problem.MemoryLimit != "64MB" {
		t.Errorf("expected memory limit '64MB', got %q", problem.MemoryLimit)
	}
	if len(problem.TestCases) != 1 {
		t.Fatalf("expected 1 test case, got %d", len(problem.TestCases))
	}
	if problem.TestCases[0].Input != "8" {
		t.Errorf("expected test case input '8', got %q", problem.TestCases[0].Input)
	}
	if problem.TestCases[0].Output != "YES" {
		t.Errorf("expected test case output 'YES', got %q", problem.TestCases[0].Output)
	}
}

func TestLeetCodeScraper_MockGraphQL(t *testing.T) {
	mockGraphQLResponse := `{
		"data": {
			"question": {
				"questionId": "1",
				"title": "Two Sum",
				"titleSlug": "two-sum",
				"content": "<p>Given an array of integers <code>nums</code> and an integer <code>target</code>, return indices of the two numbers such that they add up to target.</p><pre><strong>Input:</strong> nums = [2,7,11,15], target = 9\n<strong>Output:</strong> [0,1]</pre>",
				"difficulty": "Easy",
				"exampleTestcaseList": ["[2,7,11,15]\n9"],
				"sampleTestCase": "[2,7,11,15]\n9",
				"codeSnippets": [
					{"lang": "Python3", "langSlug": "python3", "code": "class Solution:\n    def twoSum(self, nums: List[int], target: int) -> List[int]:\n        pass"},
					{"lang": "C++", "langSlug": "cpp", "code": "class Solution {\npublic:\n    vector<int> twoSum(vector<int>& nums, int target) {\n        \n    }\n};"}
				],
				"topicTags": [{"name": "Array", "slug": "array"}, {"name": "Hash Table", "slug": "hash-table"}]
			}
		}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(mockGraphQLResponse))
	}))
	defer server.Close()

	scraper := &LeetCodeScraper{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		graphqlURL: server.URL,
	}

	problem, err := scraper.Scrape(context.Background(), "https://leetcode.com/problems/two-sum/")
	if err != nil {
		t.Fatalf("unexpected error scraping leetcode: %v", err)
	}

	if problem.Title != "Two Sum" {
		t.Errorf("expected title 'Two Sum', got %q", problem.Title)
	}
	if problem.Difficulty != "easy" {
		t.Errorf("expected difficulty 'easy', got %q", problem.Difficulty)
	}
	if len(problem.TestCases) != 1 {
		t.Fatalf("expected 1 testcase, got %d", len(problem.TestCases))
	}
	if len(problem.Templates) != 2 {
		t.Errorf("expected 2 templates, got %d", len(problem.Templates))
	}
	if _, ok := problem.Templates["py"]; !ok {
		t.Errorf("expected py template to exist")
	}
}
