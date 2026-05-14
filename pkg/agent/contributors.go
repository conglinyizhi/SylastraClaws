package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/conglinyizhi/SylastraClaws/pkg/logger"
)

// ContributorManager centralises registration of all PromptContributor
// implementations. New contributors should be added here as methods,
// not scattered across different files.
type ContributorManager struct {
	builders []*ContextBuilder
}

func NewContributorManager(builders ...*ContextBuilder) *ContributorManager {
	return &ContributorManager{builders: builders}
}

// newContributorManagerFromAgents collects all ContextBuilder instances from
// the registry and returns a ContributorManager that targets all of them.
func newContributorManagerFromAgents(registry *AgentRegistry) *ContributorManager {
	var builders []*ContextBuilder
	for _, agentID := range registry.ListAgentIDs() {
		if agent, ok := registry.GetAgent(agentID); ok && agent.ContextBuilder != nil {
			builders = append(builders, agent.ContextBuilder)
		}
	}
	return NewContributorManager(builders...)
}

// RegisterToolDiscovery registers the tool-discovery prompt contributor.
func (m *ContributorManager) RegisterToolDiscovery(useBM25, useRegex bool) {
	if !useBM25 && !useRegex {
		return
	}
	for _, cb := range m.builders {
		if err := cb.RegisterPromptContributor(toolDiscoveryPromptContributor{
			useBM25:  useBM25,
			useRegex: useRegex,
		}); err != nil {
			logger.WarnCF("agent", "Failed to register tool discovery prompt contributor",
				map[string]any{"error": err.Error()})
		}
	}
}

// RegisterMCP registers the mcpServerPromptContributor for a single server.
// toolCount <= 0 causes a no-op.
func (m *ContributorManager) RegisterMCP(
	serverName string, toolCount int, deferred, unstable bool,
) {
	if serverName == "" || toolCount <= 0 {
		return
	}
	for _, cb := range m.builders {
		if err := cb.RegisterPromptContributor(mcpServerPromptContributor{
			serverName: serverName,
			toolCount:  toolCount,
			deferred:   deferred,
			unstable:   unstable,
		}); err != nil {
			logger.WarnCF("agent", "Failed to register MCP prompt contributor",
				map[string]any{"server": serverName, "error": err.Error()})
		}
	}
}

var _ PromptContributor = (*toolListPromptContributor)(nil)

// toolListPromptContributor injects the current mission task list
// into the context layer of the system prompt.
type toolListPromptContributor struct {
	getItems func() []string
}

func (c toolListPromptContributor) PromptSource() PromptSourceDescriptor {
	return PromptSourceDescriptor{
		ID:              PromptSourceID("sylastraclaws:mission_list"),
		Owner:           "sylastraclaws",
		Description:     "Current mission task list",
		Allowed:         []PromptPlacement{{Layer: PromptLayerContext, Slot: PromptSlotRuntime}},
		StableByDefault: false,
	}
}

func (c toolListPromptContributor) ContributePrompt(
	_ context.Context, _ PromptBuildRequest,
) ([]PromptPart, error) {
	if c.getItems == nil {
		return nil, nil
	}
	items := c.getItems()
	if len(items) == 0 {
		return nil, nil
	}

	var sb strings.Builder
	sb.WriteString("Active mission items:\n")
	for _, item := range items {
		sb.WriteString("- " + item + "\n")
	}

	return []PromptPart{
		{
			ID:      "context.mission_list",
			Layer:   PromptLayerContext,
			Slot:    PromptSlotRuntime,
			Source:  PromptSource{ID: PromptSourceID("sylastraclaws:mission_list"), Name: "sylastraclaws:mission_list"},
			Title:   "Current mission tasks",
			Content: strings.TrimSpace(sb.String()),
			Stable:  false,
			Cache:   PromptCacheEphemeral,
		},
	}, nil
}

// formatToolDiscoveryRule generates the instruction block for
// tool-discovery via regex or BM25 search. Copied from the old
// toolDiscoveryPromptContributor logic; kept here because it is only
// called from this manager.
func formatToolDiscoveryRule(useBM25, useRegex bool) string {
	if !useBM25 && !useRegex {
		return ""
	}

	var b strings.Builder
	b.WriteString("To find a tool that you need, search the tool list")
	if useBM25 {
		b.WriteString(" using `tool_search_tool_bm25` with a natural language query")
	}
	if useBM25 && useRegex {
		b.WriteString(", or")
	}
	if useRegex {
		b.WriteString(" using `tool_search_tool_regex` with a regex pattern")
	}
	b.WriteString(".")
	return b.String()
}

// mcpServerPromptContributor injects MCP server availability info.
// Defined here so ContributorManager can create it without import cycles.
type mcpServerPromptContributor struct {
	serverName string
	toolCount  int
	deferred   bool
	unstable   bool
}

func (c mcpServerPromptContributor) PromptSource() PromptSourceDescriptor {
	slot := PromptSlotMCP
	if c.unstable {
		slot = PromptSlotMCPDynamic
	}
	return PromptSourceDescriptor{
		ID:              mcpPromptSourceID(c.serverName),
		Owner:           "mcp",
		Description:     fmt.Sprintf("MCP server %q capability prompt", c.serverName),
		Allowed:         []PromptPlacement{{Layer: PromptLayerCapability, Slot: slot}},
		StableByDefault: true,
	}
}

func (c mcpServerPromptContributor) ContributePrompt(
	_ context.Context,
	_ PromptBuildRequest,
) ([]PromptPart, error) {
	serverName := strings.TrimSpace(c.serverName)
	if serverName == "" || c.toolCount <= 0 {
		return nil, nil
	}

	availability := "available as native tools"
	if c.deferred {
		availability = "hidden behind tool discovery until unlocked"
	}

	slot := PromptSlotMCP
	if c.unstable {
		slot = PromptSlotMCPDynamic
	}

	return []PromptPart{
		{
			ID:     "capability.mcp." + promptSourceComponent(serverName),
			Layer:  PromptLayerCapability,
			Slot:   slot,
			Source: PromptSource{ID: mcpPromptSourceID(serverName), Name: "mcp:" + serverName},
			Title:  "MCP server capability",
			Content: fmt.Sprintf(
				"MCP server `%s` is connected. It contributes %d tool(s), currently %s.",
				serverName,
				c.toolCount,
				availability,
			),
			Stable: true,
			Cache:  PromptCacheEphemeral,
		},
	}, nil
}

func mcpPromptSourceID(serverName string) PromptSourceID {
	return PromptSourceID("mcp:" + promptSourceComponent(serverName))
}

func promptSourceComponent(value string) string {
	const maxLen = 64

	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "unnamed"
	}

	var b strings.Builder
	lastWasSep := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			lastWasSep = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastWasSep = false
		case r == '-' || r == '_':
			if !lastWasSep && b.Len() > 0 {
				b.WriteRune(r)
				lastWasSep = true
			}
		default:
			if !lastWasSep && b.Len() > 0 {
				b.WriteRune('_')
				lastWasSep = true
			}
		}
	}

	result := strings.Trim(b.String(), "_")
	if result == "" {
		return "unnamed"
	}
	if len(result) > maxLen {
		return result[:maxLen]
	}
	return result
}

// Legacy helpers kept for backward compatibility with the old
// toolDiscoveryPromptContributor referenced from context.go.
type toolDiscoveryPromptContributor struct {
	useBM25  bool
	useRegex bool
}

func (c toolDiscoveryPromptContributor) PromptSource() PromptSourceDescriptor {
	return PromptSourceDescriptor{
		ID:              PromptSourceToolDiscovery,
		Owner:           "tools",
		Description:     "Tool discovery instructions",
		Allowed:         []PromptPlacement{{Layer: PromptLayerCapability, Slot: PromptSlotTooling}},
		StableByDefault: true,
	}
}

func (c toolDiscoveryPromptContributor) ContributePrompt(
	_ context.Context,
	_ PromptBuildRequest,
) ([]PromptPart, error) {
	content := formatToolDiscoveryRule(c.useBM25, c.useRegex)
	if strings.TrimSpace(content) == "" {
		return nil, nil
	}

	return []PromptPart{
		{
			ID:      "capability.tool_discovery",
			Layer:   PromptLayerCapability,
			Slot:    PromptSlotTooling,
			Source:  PromptSource{ID: PromptSourceToolDiscovery, Name: "tool_registry:discovery"},
			Title:   "tool discovery",
			Content: content,
			Stable:  true,
			Cache:   PromptCacheEphemeral,
		},
	}, nil
}
