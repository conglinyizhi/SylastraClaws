package integrationtools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type BaiduSearchProvider struct {
	apiKey  string
	baseURL string
	proxy   string
	client  *http.Client
}

func (p *BaiduSearchProvider) Search(
	ctx context.Context,
	query string,
	count int,
	rangeCode string,
) (string, error) {
	if p.apiKey == "" {
		return "", errors.New("no API key provided")
	}

	searchURL := p.baseURL
	if searchURL == "" {
		searchURL = "https://qianfan.baidubce.com/v2/ai_search/web_search"
	}

	payload := map[string]any{
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": query,
			},
		},
		"search_source":        "baidu_search_v2",
		"resource_type_filter": []map[string]any{{"type": "web", "top_k": count}},
	}
	if recencyFilter := mapBaiduRecencyFilter(rangeCode); recencyFilter != "" {
		payload["search_recency_filter"] = recencyFilter
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", searchURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("baidu search request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("baidu search API error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		References []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
		} `json:"references"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.References) == 0 {
		return fmt.Sprintf("No results for: %s", query), nil
	}

	lines := []string{fmt.Sprintf("Results for: %s (via Baidu Search)", query)}
	for i, item := range result.References {
		if i >= count {
			break
		}
		lines = append(lines, fmt.Sprintf("%d. %s\n   %s", i+1, item.Title, item.URL))
		if item.Content != "" {
			lines = append(lines, fmt.Sprintf("   %s", item.Content))
		}
	}

	return strings.Join(lines, "\n"), nil
}
