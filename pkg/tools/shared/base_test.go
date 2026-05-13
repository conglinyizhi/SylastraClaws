package toolshared

import (
	"context"
	"testing"

	"github.com/sipeed/picoclaw/pkg/session"
)

func TestPromptMetadataDefaults(t *testing.T) {
	pm := PromptMetadata{}
	if pm.Layer != "" {
		t.Errorf("expected empty Layer, got %q", pm.Layer)
	}
	if pm.Slot != "" {
		t.Errorf("expected empty Slot, got %q", pm.Slot)
	}
	if pm.Source != "" {
		t.Errorf("expected empty Source, got %q", pm.Source)
	}
}

func TestPromptMetadataConstants(t *testing.T) {
	tests := []struct {
		got, want string
		name      string
	}{
		{ToolPromptLayerCapability, "capability", "LayerCapability"},
		{ToolPromptSlotTooling, "tooling", "SlotTooling"},
		{ToolPromptSlotMCP, "mcp", "SlotMCP"},
		{ToolPromptSourceRegistry, "tool_registry:native", "SourceRegistry"},
		{ToolPromptSourceDiscovery, "tool_registry:discovery", "SourceDiscovery"},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.want)
		}
	}
}

func TestPromptMetadataProviderInterface(t *testing.T) {
	// Ensure a simple struct can satisfy the interface.
	var _ PromptMetadataProvider = mockPromptMeta{}

	// Verify the returned metadata matches.
	p := mockPromptMeta{}
	pm := p.PromptMetadata()
	if pm.Layer != "test-layer" || pm.Slot != "test-slot" || pm.Source != "test-source" {
		t.Fatalf("unexpected PromptMetadata: %+v", pm)
	}
}

type mockPromptMeta struct{}

func (m mockPromptMeta) PromptMetadata() PromptMetadata {
	return PromptMetadata{
		Layer:  "test-layer",
		Slot:   "test-slot",
		Source: "test-source",
	}
}

func TestWithToolContextAndGet(t *testing.T) {
	ctx := context.Background()
	channel := "test-channel-1"
	chatID := "test-chat-1"

	ctx = WithToolContext(ctx, channel, chatID)

	if got := ToolChannel(ctx); got != channel {
		t.Errorf("ToolChannel = %q, want %q", got, channel)
	}
	if got := ToolChatID(ctx); got != chatID {
		t.Errorf("ToolChatID = %q, want %q", got, chatID)
	}
}

func TestToolContextDefaultsEmpty(t *testing.T) {
	ctx := context.Background()

	if got := ToolChannel(ctx); got != "" {
		t.Errorf("ToolChannel on bare ctx = %q, want empty", got)
	}
	if got := ToolChatID(ctx); got != "" {
		t.Errorf("ToolChatID on bare ctx = %q, want empty", got)
	}
	if got := ToolMessageID(ctx); got != "" {
		t.Errorf("ToolMessageID on bare ctx = %q, want empty", got)
	}
	if got := ToolReplyToMessageID(ctx); got != "" {
		t.Errorf("ToolReplyToMessageID on bare ctx = %q, want empty", got)
	}
	if got := ToolAgentID(ctx); got != "" {
		t.Errorf("ToolAgentID on bare ctx = %q, want empty", got)
	}
	if got := ToolSessionKey(ctx); got != "" {
		t.Errorf("ToolSessionKey on bare ctx = %q, want empty", got)
	}
	if got := ToolSessionScope(ctx); got != nil {
		t.Errorf("ToolSessionScope on bare ctx = %v, want nil", got)
	}
}

func TestWithToolMessageContext(t *testing.T) {
	ctx := context.Background()
	msgID := "msg-42"
	replyToID := "msg-17"

	ctx = WithToolMessageContext(ctx, msgID, replyToID)

	if got := ToolMessageID(ctx); got != msgID {
		t.Errorf("ToolMessageID = %q, want %q", got, msgID)
	}
	if got := ToolReplyToMessageID(ctx); got != replyToID {
		t.Errorf("ToolReplyToMessageID = %q, want %q", got, replyToID)
	}
}

func TestWithToolInboundContext(t *testing.T) {
	ctx := context.Background()
	channel := "chan-1"
	chatID := "chat-1"
	msgID := "msg-99"
	replyTo := "msg-88"

	ctx = WithToolInboundContext(ctx, channel, chatID, msgID, replyTo)

	if got := ToolChannel(ctx); got != channel {
		t.Errorf("ToolChannel = %q, want %q", got, channel)
	}
	if got := ToolChatID(ctx); got != chatID {
		t.Errorf("ToolChatID = %q, want %q", got, chatID)
	}
	if got := ToolMessageID(ctx); got != msgID {
		t.Errorf("ToolMessageID = %q, want %q", got, msgID)
	}
	if got := ToolReplyToMessageID(ctx); got != replyTo {
		t.Errorf("ToolReplyToMessageID = %q, want %q", got, replyTo)
	}
}

func TestWithToolSessionContext(t *testing.T) {
	ctx := context.Background()
	agentID := "agent-alpha"
	sessionKey := "sk-abc123"
	scope := &session.SessionScope{
		Version:    1,
		AgentID:    "agent-alpha",
		Channel:    "telegram",
		Account:    "test-account",
		Dimensions: []string{"dim1", "dim2"},
		Values:     map[string]string{"dim1": "val1"},
	}

	ctx = WithToolSessionContext(ctx, agentID, sessionKey, scope)

	if got := ToolAgentID(ctx); got != agentID {
		t.Errorf("ToolAgentID = %q, want %q", got, agentID)
	}
	if got := ToolSessionKey(ctx); got != sessionKey {
		t.Errorf("ToolSessionKey = %q, want %q", got, sessionKey)
	}

	gotScope := ToolSessionScope(ctx)
	if gotScope == nil {
		t.Fatal("ToolSessionScope returned nil")
	}
	if gotScope.Version != scope.Version {
		t.Errorf("scope.Version = %d, want %d", gotScope.Version, scope.Version)
	}
	if gotScope.AgentID != scope.AgentID {
		t.Errorf("scope.AgentID = %q, want %q", gotScope.AgentID, scope.AgentID)
	}
	if gotScope.Channel != scope.Channel {
		t.Errorf("scope.Channel = %q, want %q", gotScope.Channel, scope.Channel)
	}
	if gotScope.Account != scope.Account {
		t.Errorf("scope.Account = %q, want %q", gotScope.Account, scope.Account)
	}
	if len(gotScope.Dimensions) != len(scope.Dimensions) {
		t.Errorf("len(Dimensions) = %d, want %d", len(gotScope.Dimensions), len(scope.Dimensions))
	}
	if gotScope.Values["dim1"] != "val1" {
		t.Errorf("scope.Values[\"dim1\"] = %q, want %q", gotScope.Values["dim1"], "val1")
	}
}

func TestToolSessionScopeCloneIsolation(t *testing.T) {
	ctx := context.Background()
	scope := &session.SessionScope{
		Version:    1,
		AgentID:    "agent-1",
		Channel:    "discord",
		Dimensions: []string{"dim1"},
		Values:     map[string]string{"dim1": "original"},
	}

	ctx = WithToolSessionContext(ctx, "agent-1", "sk-xyz", scope)

	// Mutate the original after setting it on context.
	scope.Dimensions[0] = "mutated"
	scope.Values["dim1"] = "mutated"

	gotScope := ToolSessionScope(ctx)
	if gotScope.Dimensions[0] != "dim1" {
		t.Errorf("tool scope dimensions mutated via original, got %q", gotScope.Dimensions[0])
	}
	if gotScope.Values["dim1"] != "original" {
		t.Errorf("tool scope values mutated via original, got %q", gotScope.Values["dim1"])
	}
}

func TestToolSessionScopeNil(t *testing.T) {
	ctx := context.Background()
	ctx = WithToolSessionContext(ctx, "agent-1", "sk-xyz", nil)

	gotScope := ToolSessionScope(ctx)
	if gotScope != nil {
		t.Errorf("ToolSessionScope with nil input = %v, want nil", gotScope)
	}
}

func TestToolToSchema(t *testing.T) {
	tool := &mockTool{
		name:        "test_tool",
		description: "Test tool description",
		params: map[string]any{
			"type":        "object",
			"properties":  map[string]any{},
			"required":    []string{},
		},
	}

	schema := ToolToSchema(tool)

	if schema["type"] != "function" {
		t.Errorf("schema.type = %q, want %q", schema["type"], "function")
	}

	fn, ok := schema["function"].(map[string]any)
	if !ok {
		t.Fatal("schema['function'] is not a map")
	}
	if fn["name"] != "test_tool" {
		t.Errorf("function.name = %q, want %q", fn["name"], "test_tool")
	}
	if fn["description"] != "Test tool description" {
		t.Errorf("function.description = %q, want %q", fn["description"], "Test tool description")
	}
}

type mockTool struct {
	name        string
	description string
	params      map[string]any
}

func (m *mockTool) Name() string                                { return m.name }
func (m *mockTool) Description() string                         { return m.description }
func (m *mockTool) Parameters() map[string]any                  { return m.params }
func (m *mockTool) Execute(_ context.Context, _ map[string]any) *ToolResult { return NewToolResult("ok") }

func TestToolInterfaceSatisfied(t *testing.T) {
	var _ Tool = &mockTool{}
}

func TestAsyncCallbackSignature(t *testing.T) {
	// Just verify the type compiles:
	var cb AsyncCallback = func(_ context.Context, _ *ToolResult) {}
	_ = cb
}

func TestAsyncExecutorInterface(t *testing.T) {
	// Verify that a tool can implement AsyncExecutor.
	var _ AsyncExecutor = &mockAsyncTool{}
}

type mockAsyncTool struct{ mockTool }

func (m *mockAsyncTool) ExecuteAsync(ctx context.Context, args map[string]any, cb AsyncCallback) *ToolResult {
	result := m.Execute(ctx, args)
	return result
}
