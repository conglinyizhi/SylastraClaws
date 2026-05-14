package integrationtools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type SearXNGSearchProvider struct {
	baseURL string
	proxy   string
	client  *http.Client
}

func (p *SearXNGSearchProvider) Search(
	ctx context.Context,
	query string,
	count int,
	rangeCode string,
) (string, error) {
	if p.baseURL == "" {
		return "", errors.New("no SearXNG URL provided")
	}

	searchURL := fmt.Sprintf("%s/search?q=%s&format=json&categories=general",
		strings.TrimSuffix(p.baseURL, "/"),
		url.QueryEscape(query))
	if timeRange := mapSearXNGTimeRange(rangeCode); timeRange != "" {
		searchURL += "&time_range=" + url.QueryEscape(timeRange)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	client := p.client
	if client == nil {
		client = &http.Client{Timeout: searchTimeout}
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("SearXNG returned status %d", resp.StatusCode)
	}

	var result struct {
		Results []struct {
			Title   string  `json:"title"`
			URL     string  `json:"url"`
			Content string  `json:"content"`
			Engine  string  `json:"engine"`
			Score   float64 `json:"score"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Results) == 0 {
		return fmt.Sprintf("No results for: %s", query), nil
	}

	// Limit results to requested count
	if len(result.Results) > count {
		result.Results = result.Results[:count]
	}

	// Format results in standard PicoClaw format
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Results for: %s (via SearXNG)\n", query))
	for i, r := range result.Results {
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, r.Title))
		b.WriteString(fmt.Sprintf("   %s\n", r.URL))
		if r.Content != "" {
			b.WriteString(fmt.Sprintf("   %s\n", r.Content))
		}
	}

	return b.String(), nil
}
