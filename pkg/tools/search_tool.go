package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/conglinyizhi/SylastraClaws/pkg/logger"
	"github.com/conglinyizhi/SylastraClaws/pkg/utils"
)

const (
	MaxRegexPatternLength = 200
)

type ToolSearchTool struct {
	registry         *ToolRegistry
	ttl              int
	maxSearchResults int

	// BM25 cache: rebuilt only when the registry version changes.
	cacheMu      sync.Mutex
	cachedEngine *bm25CachedEngine
	cacheVersion uint64
}

func NewToolSearchTool(r *ToolRegistry, ttl int, maxSearchResults int) *ToolSearchTool {
	return &ToolSearchTool{registry: r, ttl: ttl, maxSearchResults: maxSearchResults}
}

func (t *ToolSearchTool) Name() string {
	return "tool_search"
}

func (t *ToolSearchTool) Description() string {
	return "Search available hidden tools on-demand. When the query is a valid regex pattern, searches both tool names and descriptions as regex (most precise). Otherwise, uses BM25 natural language search. Returns JSON schemas of discovered tools."
}

func (t *ToolSearchTool) PromptMetadata() PromptMetadata {
	return PromptMetadata{
		Layer:  ToolPromptLayerCapability,
		Slot:   ToolPromptSlotTooling,
		Source: ToolPromptSourceDiscovery,
	}
}

func (t *ToolSearchTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Search query. If it's a valid regex pattern, searches by regex (name + description). Otherwise, searches by BM25 natural language match.",
			},
		},
		"required": []string{"query"},
	}
}

func (t *ToolSearchTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	query, ok := args["query"].(string)
	if !ok || strings.TrimSpace(query) == "" {
		return ErrorResult("Missing or invalid 'query' argument. Must be a non-empty string.")
	}

	query = strings.TrimSpace(query)

	// Try regex first. If valid regex, use it.
	if _, err := regexp.Compile("(?i)" + query); err == nil {
		if len(query) > MaxRegexPatternLength {
			logger.WarnCF("discovery", "Regex pattern rejected (too long)", map[string]any{"len": len(query)})
			// Fall through to BM25 instead of erroring out.
		} else {
			res, err := t.registry.SearchRegex(query, t.maxSearchResults)
			if err == nil {
				logger.InfoCF("discovery", "Regex search completed", map[string]any{"query": query, "results": len(res)})
				return formatDiscoveryResponse(t.registry, res, t.ttl)
			}
			// Invalid regex — log and fall through to BM25.
			logger.DebugCF("discovery", "Regex failed, falling back to BM25", map[string]any{"query": query, "error": err.Error()})
		}
	}

	// Fallback: BM25 natural language search.
	logger.DebugCF("discovery", "BM25 search", map[string]any{"query": query})
	cached := t.getOrBuildEngine()
	if cached == nil {
		logger.DebugCF("discovery", "BM25 search: no hidden tools available", nil)
		return SilentResult("No tools found matching the query.")
	}
	ranked := cached.engine.Search(query, t.maxSearchResults)
	if len(ranked) == 0 {
		logger.DebugCF("discovery", "BM25 search: no matches", map[string]any{"query": query})
		return SilentResult("No tools found matching the query.")
	}

	results := make([]ToolSearchResult, len(ranked))
	for i, r := range ranked {
		results[i] = ToolSearchResult{
			Name:        r.Document.Name,
			Description: r.Document.Description,
		}
	}
	logger.InfoCF("discovery", "BM25 search completed", map[string]any{"query": query, "results": len(results)})
	return formatDiscoveryResponse(t.registry, results, t.ttl)
}

// ToolSearchResult represents the result returned to the LLM.
type ToolSearchResult struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (r *ToolRegistry) SearchRegex(pattern string, maxSearchResults int) ([]ToolSearchResult, error) {
	if maxSearchResults <= 0 {
		return nil, nil
	}
	regex, err := regexp.Compile("(?i)" + pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to compile regex pattern %q: %w", pattern, err)
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	var results []ToolSearchResult
	for _, name := range r.sortedToolNames() {
		entry := r.tools[name]
		if !entry.IsCore {
			desc := entry.Tool.Description()
			if regex.MatchString(name) || regex.MatchString(desc) {
				results = append(results, ToolSearchResult{Name: name, Description: desc})
				if len(results) >= maxSearchResults {
					break
				}
			}
		}
	}
	return results, nil
}

func formatDiscoveryResponse(registry *ToolRegistry, results []ToolSearchResult, ttl int) *ToolResult {
	if len(results) == 0 {
		return SilentResult("No tools found matching the query.")
	}
	names := make([]string, len(results))
	for i, r := range results {
		names[i] = r.Name
	}
	registry.PromoteTools(names, ttl)
	logger.InfoCF("discovery", "Promoted tools", map[string]any{"tools": names, "ttl": ttl})
	b, err := json.Marshal(results)
	if err != nil {
		return ErrorResult("Failed to format search results: " + err.Error())
	}
	msg := fmt.Sprintf(
		"Found %d tools:\n%s\n\nSUCCESS: These tools have been temporarily UNLOCKED as native tools! In your next response, you can call them directly just like any normal tool",
		len(results),
		string(b),
	)
	return SilentResult(msg)
}

// searchDoc — lightweight internal type for BM25 corpus.
type searchDoc struct {
	Name        string
	Description string
}

// bm25CachedEngine wraps a BM25Engine with its corpus snapshot.
type bm25CachedEngine struct {
	engine *utils.BM25Engine[searchDoc]
}

func snapshotToSearchDocs(snap HiddenToolSnapshot) []searchDoc {
	docs := make([]searchDoc, len(snap.Docs))
	for i, d := range snap.Docs {
		docs[i] = searchDoc{Name: d.Name, Description: d.Description}
	}
	return docs
}

func buildBM25Engine(docs []searchDoc) *utils.BM25Engine[searchDoc] {
	return utils.NewBM25Engine(
		docs,
		func(doc searchDoc) string { return doc.Name + " " + doc.Description },
	)
}

func (t *ToolSearchTool) getOrBuildEngine() *bm25CachedEngine {
	if t.cachedEngine != nil && t.cacheVersion == t.registry.Version() {
		return t.cachedEngine
	}
	t.cacheMu.Lock()
	defer t.cacheMu.Unlock()
	snap := t.registry.SnapshotHiddenTools()
	if t.cachedEngine != nil && t.cacheVersion == snap.Version {
		return t.cachedEngine
	}
	docs := snapshotToSearchDocs(snap)
	if len(docs) == 0 {
		t.cachedEngine = nil
		t.cacheVersion = snap.Version
		return nil
	}
	cached := &bm25CachedEngine{engine: buildBM25Engine(docs)}
	t.cachedEngine = cached
	t.cacheVersion = snap.Version
	logger.DebugCF("discovery", "BM25 engine rebuilt", map[string]any{"docs": len(docs), "version": snap.Version})
	return cached
}

// SearchBM25 — non-cached variant for tests and one-off use.
func (r *ToolRegistry) SearchBM25(query string, maxSearchResults int) []ToolSearchResult {
	snap := r.SnapshotHiddenTools()
	docs := snapshotToSearchDocs(snap)
	if len(docs) == 0 {
		return nil
	}
	ranked := buildBM25Engine(docs).Search(query, maxSearchResults)
	if len(ranked) == 0 {
		return nil
	}
	out := make([]ToolSearchResult, len(ranked))
	for i, r := range ranked {
		out[i] = ToolSearchResult{Name: r.Document.Name, Description: r.Document.Description}
	}
	return out
}
