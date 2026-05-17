package tools

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"regexp"
	"strings"

	"github.com/conglinyizhi/SylastraClaws/pkg/logger"
	integrationtools "github.com/conglinyizhi/SylastraClaws/pkg/tools/integration"
)

// urlLikePattern matches inputs that look like URLs or host:path constructs.
var urlLikePattern = regexp.MustCompile(`^(https?://|[a-zA-Z0-9]([-a-zA-Z0-9]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([-a-zA-Z0-9]*[a-zA-Z0-9])?)*(:[0-9]+)?(/|$)`)

// WebSearchAndGetTool merges web_search and web_fetch into a single tool.
type WebSearchAndGetTool struct {
	searchTool *integrationtools.WebSearchTool
	fetchTool  *integrationtools.WebFetchTool
}

func NewWebSearchAndGetTool(searchOpts integrationtools.WebSearchToolOptions, proxy, format string, fetchLimitBytes int64, privateHostWhitelist []string) (*WebSearchAndGetTool, error) {
	searchTool, err := integrationtools.NewWebSearchTool(searchOpts)
	if err != nil {
		return nil, fmt.Errorf("web_search_and_get: %w", err)
	}
	fetchTool, err := integrationtools.NewWebFetchToolWithConfig(50000, proxy, format, fetchLimitBytes, privateHostWhitelist)
	if err != nil {
		logger.WarnCF("tool", "web_search_and_get: web_fetch not available, search-only", nil)
		fetchTool = nil
	}
	return &WebSearchAndGetTool{searchTool: searchTool, fetchTool: fetchTool}, nil
}

func (t *WebSearchAndGetTool) Name() string {
	return "web_search_and_get"
}

func (t *WebSearchAndGetTool) Description() string {
	return "Search the web or fetch a URL. If input looks like a URL (domain, domain/path, IP:port), fetches page content. Otherwise, performs web search. Use try_curl=true to also attempt curl/wget as fetch fallback."
}

func (t *WebSearchAndGetTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Search query, or a URL (domain, https://host/path, ip:port/path, etc.) to fetch",
			},
			"count": map[string]any{
				"type":        "integer",
				"description": "Number of search results (default: 5, max: 10; only used for search)",
				"minimum":     1.0,
				"maximum":     10.0,
			},
			"range": map[string]any{
				"type":        "string",
				"description": "Time filter for search: d (day), w (week), m (month), y (year)",
				"enum":        []string{"d", "w", "m", "y"},
			},
			"try_curl": map[string]any{
				"type":        "boolean",
				"description": "If URL fetch fails, try curl/wget as fallback (default: false)",
			},
			"max_chars": map[string]any{
				"type":        "integer",
				"description": "Max chars when fetching a URL (default: 50000)",
				"minimum":     100.0,
			},
		},
		"required": []string{"query"},
	}
}

func (t *WebSearchAndGetTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	query, ok := args["query"].(string)
	if !ok || strings.TrimSpace(query) == "" {
		return ErrorResult("query is required")
	}
	query = strings.TrimSpace(query)
	tryCurl, _ := args["try_curl"].(bool)

	if looksLikeURL(query) {
		return t.tryFetch(ctx, query, args, tryCurl)
	}

	if t.searchTool == nil {
		return ErrorResult("search provider not configured")
	}
	return t.searchTool.Execute(ctx, args)
}

func looksLikeURL(s string) bool {
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		return true
	}
	return urlLikePattern.MatchString(s)
}

func (t *WebSearchAndGetTool) tryFetch(ctx context.Context, rawURL string, args map[string]any, tryCurl bool) *ToolResult {
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" {
		if tryCurl {
			return t.curlFetch(ctx, rawURL, args)
		}
		return ErrorResult(fmt.Sprintf("invalid URL: %v", err))
	}

	if t.fetchTool != nil {
		result := t.fetchTool.Execute(ctx, args)
		if result != nil && result.IsError && tryCurl {
			return t.curlFetch(ctx, rawURL, args)
		}
		if result != nil && !result.IsError {
			return result
		}
	}

	if tryCurl {
		return t.curlFetch(ctx, rawURL, args)
	}
	return ErrorResult("URL fetch not available; enable web_fetch config or set try_curl=true")
}

func (t *WebSearchAndGetTool) curlFetch(_ context.Context, rawURL string, args map[string]any) *ToolResult {
	maxChars := 50000
	if mc, ok := args["max_chars"].(float64); ok && int(mc) > 100 {
		maxChars = int(mc)
	}

	var body string
	var tool string

	if _, err := exec.LookPath("curl"); err == nil {
		logger.DebugCF("tool", "web_search_and_get: trying curl", map[string]any{"url": rawURL})
		cmd := exec.Command("curl", "-sSL", "--max-time", "30", rawURL)
		out, err := cmd.Output()
		if err == nil {
			body = string(out)
			tool = "curl"
		}
	}

	if body == "" {
		if _, err := exec.LookPath("wget"); err == nil {
			logger.DebugCF("tool", "web_search_and_get: trying wget", map[string]any{"url": rawURL})
			cmd := exec.Command("wget", "-q", "-O", "-", "--timeout=30", rawURL)
			out, err := cmd.Output()
			if err == nil {
				body = string(out)
				tool = "wget"
			}
		}
	}

	if body == "" {
		return ErrorResult("failed to fetch URL via curl or wget")
	}

	if len(body) > maxChars {
		body = body[:maxChars]
	}

	logger.InfoCF("tool", "web_search_and_get: fetched via "+tool, map[string]any{"url": rawURL, "chars": len(body)})
	return SilentResult(fmt.Sprintf("Content from %s via %s:\n\n%s", rawURL, tool, body))
}
