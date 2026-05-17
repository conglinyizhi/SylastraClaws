package agent

import (
	"context"
	"fmt"
	"sort"
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
		if err := cb.RegisterPromptContributor(toolDiscoveryPromptContributor{}); err != nil {
			logger.WarnCF("agent", "Failed to register tool discovery prompt contributor",
				map[string]any{"error": err.Error()})
		}
	}
	}
// RegisterSubagentDirectory registers a contributor that injects the list of
// available sub-agents (agents that can be spawned) into the system prompt.
// The getter is called per-turn so the list reflects the latest config.
func (m *ContributorManager) RegisterSubagentDirectory(getter func() map[string]string) {
	if getter == nil {
		return
	}
	for _, cb := range m.builders {
		if err := cb.RegisterPromptContributor(subagentDirectoryContributor{
			getter: getter,
		}); err != nil {
			logger.WarnCF("agent", "Failed to register sub-agent directory contributor",
				map[string]any{"error": err.Error()})
		}
	}
}

// subagentDirectoryContributor injects the list of available sub-agents
// into the capability layer of the system prompt.
type subagentDirectoryContributor struct {
	getter func() map[string]string
}

func (c subagentDirectoryContributor) PromptSource() PromptSourceDescriptor {
	return PromptSourceDescriptor{
		ID:              PromptSourceID("sylastraclaws:subagent_directory"),
		Owner:           "sylastraclaws",
		Description:     "Available sub-agent directory for task delegation",
		Allowed:         []PromptPlacement{{Layer: PromptLayerCapability, Slot: PromptSlotTooling}},
		StableByDefault: true,
	}
}

func (c subagentDirectoryContributor) ContributePrompt(
	_ context.Context, _ PromptBuildRequest,
) ([]PromptPart, error) {
	if c.getter == nil {
		return nil, nil
	}
	agents := c.getter()
	if len(agents) == 0 {
		return nil, nil
	}

	var sb strings.Builder
	sb.WriteString("Available sub-agents for task delegation.\n")
	sb.WriteString("When a task requires a specialist skill, use the `spawn` or `subagent` tool ")
	sb.WriteString("to delegate the task to the appropriate sub-agent.\n")
	sb.WriteString("If no sub-agent is suitable, complete the task yourself.\n\n")
	sb.WriteString("Sub-agents:\n")
	// Sort for deterministic order
	ids := make([]string, 0, len(agents))
	for id := range agents {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		sb.WriteString("- **" + id + "**: " + agents[id] + "\n")
	}

	return []PromptPart{
		{
			ID:      "capability.subagent_directory",
			Layer:   PromptLayerCapability,
			Slot:    PromptSlotTooling,
			Source:  PromptSource{ID: PromptSourceID("sylastraclaws:subagent_directory"), Name: "sylastraclaws:subagent_directory"},
			Title:   "Sub-agent delegation",
			Content: strings.TrimSpace(sb.String()),
			Stable:  true,
			Cache:   PromptCacheEphemeral,
		},
	}, nil
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
// formatToolDiscoveryRule generates a concise prompt rule telling the LLM how to find
// hidden tools via the combined tool_search tool.
func formatToolDiscoveryRule() string {
	var b strings.Builder
	b.WriteString("To find a hidden tool that you need, use `tool_search` with a search query. ")
	b.WriteString("If your query is a valid regex pattern, it will match tool names and descriptions precisely. ")
	b.WriteString("Otherwise, it uses natural language (BM25) search to find relevant tools.")
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

// toolDiscoveryPromptContributor generates a concise prompt rule telling the LLM
// how to find hidden tools via the combined tool_search tool.
type toolDiscoveryPromptContributor struct{}

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
	content := formatToolDiscoveryRule()
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
