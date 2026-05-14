package integrationtools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type SogouSearchProvider struct {
	proxy  string
	client *http.Client
}

func (p *SogouSearchProvider) Search(
	ctx context.Context,
	query string,
	count int,
	rangeCode string,
) (string, error) {
	const sogouWAPURL = "https://wap.sogou.com/web/searchList.jsp"

	results := make([]SearchResultItem, 0, count)
	seenURLs := make(map[string]bool)
	maxPages := min(3, (count+1)/2+1)

	for page := 1; page <= maxPages && len(results) < count; page++ {
		params := url.Values{}
		params.Set("keyword", applySogouRangeHint(query, rangeCode))
		params.Set("v", "5")
		params.Set("p", fmt.Sprintf("%d", page))

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, sogouWAPURL+"?"+params.Encode(), nil)
		if err != nil {
			return "", fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("User-Agent", sogouUserAgent)

		resp, err := p.client.Do(req)
		if err != nil {
			return "", fmt.Errorf("request failed: %w", err)
		}

		body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		resp.Body.Close()
		if err != nil {
			return "", fmt.Errorf("failed to read response: %w", err)
		}
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("Sogou returned status %d", resp.StatusCode)
		}

		html := string(body)
		if len(html) < 200 {
			break
		}

		matches := reSogouTitle.FindAllStringSubmatch(html, -1)
		for _, match := range matches {
			if len(match) < 3 {
				continue
			}

			title := stripTags(match[2])
			link := extractSogouURL(match[1])
			if title == "" || link == "" || seenURLs[link] {
				continue
			}
			seenURLs[link] = true

			start := strings.Index(html, match[0])
			snippet := ""
			if start >= 0 {
				after := html[start+len(match[0]):]
				if len(after) > 2000 {
					after = after[:2000]
				}
				if snippetMatch := reSogouSnippet.FindStringSubmatch(after); len(snippetMatch) > 1 {
					snippet = stripTags(snippetMatch[1])
				}
			}

			results = append(results, SearchResultItem{
				Title:   title,
				URL:     link,
				Snippet: snippet,
			})
			if len(results) >= count {
				break
			}
		}
	}

	if len(results) == 0 {
		return fmt.Sprintf("No results for: %s", query), nil
	}

	lines := []string{fmt.Sprintf("Results for: %s (via Sogou)", query)}
	for i, item := range results {
		lines = append(lines, fmt.Sprintf("%d. %s\n   %s", i+1, item.Title, item.URL))
		if item.Snippet != "" {
			lines = append(lines, fmt.Sprintf("   %s", item.Snippet))
		}
	}
	return strings.Join(lines, "\n"), nil
}
