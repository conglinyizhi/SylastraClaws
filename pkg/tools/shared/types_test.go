package toolshared

import (
	"context"
	"encoding/json"
	"testing"
)

func TestMessageDefaults(t *testing.T) {
	m := Message{}
	if m.Role != "" {
		t.Errorf("Role = %q, want empty", m.Role)
	}
	if m.Content != "" {
		t.Errorf("Content = %q, want empty", m.Content)
	}
	if len(m.ToolCalls) != 0 {
		t.Errorf("len(ToolCalls) = %d, want 0", len(m.ToolCalls))
	}
	if m.ToolCallID != "" {
		t.Errorf("ToolCallID = %q, want empty", m.ToolCallID)
	}
}

func TestMessageJSONRoundTrip(t *testing.T) {
	original := Message{
		Role:    "assistant",
		Content: "Hello!",
		ToolCalls: []ToolCall{
			{
				ID:   "call_1",
				Type: "function",
				Function: &FunctionCall{
					Name:      "test_func",
					Arguments: `{"key":"value"}`,
				},
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.Role != original.Role {
		t.Errorf("Role = %q, want %q", decoded.Role, original.Role)
	}
	if decoded.Content != original.Content {
		t.Errorf("Content = %q, want %q", decoded.Content, original.Content)
	}
	if len(decoded.ToolCalls) != 1 {
		t.Fatalf("len(ToolCalls) = %d, want 1", len(decoded.ToolCalls))
	}
	if decoded.ToolCalls[0].ID != "call_1" {
		t.Errorf("ToolCalls[0].ID = %q, want %q", decoded.ToolCalls[0].ID, "call_1")
	}
	if decoded.ToolCalls[0].Type != "function" {
		t.Errorf("ToolCalls[0].Type = %q, want %q", decoded.ToolCalls[0].Type, "function")
	}
	if decoded.ToolCalls[0].Function == nil {
		t.Fatal("ToolCalls[0].Function should not be nil")
	}
	if decoded.ToolCalls[0].Function.Name != "test_func" {
		t.Errorf("Function.Name = %q, want %q", decoded.ToolCalls[0].Function.Name, "test_func")
	}
	if decoded.ToolCalls[0].Function.Arguments != `{"key":"value"}` {
		t.Errorf("Function.Arguments = %q, want %q", decoded.ToolCalls[0].Function.Arguments, `{"key":"value"}`)
	}
}

func TestToolCallOptionalFields(t *testing.T) {
	m := Message{
		Role:       "tool",
		Content:    "result",
		ToolCallID: "call_1",
	}

	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.Role != "tool" {
		t.Errorf("Role = %q, want %q", decoded.Role, "tool")
	}
	if decoded.Content != "result" {
		t.Errorf("Content = %q, want %q", decoded.Content, "result")
	}
	if decoded.ToolCallID != "call_1" {
		t.Errorf("ToolCallID = %q, want %q", decoded.ToolCallID, "call_1")
	}
}

func TestMessageWithoutToolCallsOmitsField(t *testing.T) {
	m := Message{
		Role:    "user",
		Content: "hi",
	}

	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if _, ok := result["tool_calls"]; ok {
		t.Error("tool_calls should be omitted when empty (omitempty)")
	}
}

func TestToolCallOmitEmpty(t *testing.T) {
	tc := ToolCall{}
	data, err := json.Marshal(tc)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	// ID and Type have no omitempty — always present even when empty
	if _, ok := result["id"]; !ok {
		t.Error("id should always be present (no omitempty)")
	}
	if _, ok := result["type"]; !ok {
		t.Error("type should always be present (no omitempty)")
	}
	// Function has omitempty — should be missing when nil
	if _, ok := result["function"]; ok {
		t.Error("function should be omitted when nil (omitempty)")
	}
}

func TestFunctionCallJSON(t *testing.T) {
	fc := FunctionCall{
		Name:      "search",
		Arguments: `{"query":"test"}`,
	}

	data, err := json.Marshal(fc)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded FunctionCall
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.Name != "search" {
		t.Errorf("Name = %q, want %q", decoded.Name, "search")
	}
	if decoded.Arguments != `{"query":"test"}` {
		t.Errorf("Arguments = %q, want %q", decoded.Arguments, `{"query":"test"}`)
	}
}

func TestLLMResponseDefaults(t *testing.T) {
	r := LLMResponse{}
	if r.Content != "" {
		t.Errorf("Content = %q, want empty", r.Content)
	}
	if r.FinishReason != "" {
		t.Errorf("FinishReason = %q, want empty", r.FinishReason)
	}
	if len(r.ToolCalls) != 0 {
		t.Errorf("len(ToolCalls) = %d, want 0", len(r.ToolCalls))
	}
	if r.Usage != nil {
		t.Error("Usage should be nil")
	}
}

func TestLLMResponseJSONRoundTrip(t *testing.T) {
	original := LLMResponse{
		Content:      "I'll search for that",
		FinishReason: "tool_calls",
		ToolCalls: []ToolCall{
			{
				ID:   "call_1",
				Type: "function",
				Function: &FunctionCall{
					Name:      "search_web",
					Arguments: `{"q":"hello"}`,
				},
			},
		},
		Usage: &UsageInfo{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded LLMResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.Content != original.Content {
		t.Errorf("Content = %q, want %q", decoded.Content, original.Content)
	}
	if decoded.FinishReason != original.FinishReason {
		t.Errorf("FinishReason = %q, want %q", decoded.FinishReason, original.FinishReason)
	}
	if len(decoded.ToolCalls) != 1 {
		t.Fatalf("len(ToolCalls) = %d, want 1", len(decoded.ToolCalls))
	}
	if decoded.Usage == nil {
		t.Fatal("Usage should not be nil")
	}
	if decoded.Usage.PromptTokens != 10 {
		t.Errorf("Usage.PromptTokens = %d, want 10", decoded.Usage.PromptTokens)
	}
	if decoded.Usage.CompletionTokens != 20 {
		t.Errorf("Usage.CompletionTokens = %d, want 20", decoded.Usage.CompletionTokens)
	}
	if decoded.Usage.TotalTokens != 30 {
		t.Errorf("Usage.TotalTokens = %d, want 30", decoded.Usage.TotalTokens)
	}
}

func TestUsageInfoDefaults(t *testing.T) {
	u := UsageInfo{}
	if u.PromptTokens != 0 {
		t.Errorf("PromptTokens = %d, want 0", u.PromptTokens)
	}
	if u.CompletionTokens != 0 {
		t.Errorf("CompletionTokens = %d, want 0", u.CompletionTokens)
	}
	if u.TotalTokens != 0 {
		t.Errorf("TotalTokens = %d, want 0", u.TotalTokens)
	}
}

func TestToolDefinitionJSON(t *testing.T) {
	td := ToolDefinition{
		Type: "function",
		Function: ToolFunctionDefinition{
			Name:        "my_tool",
			Description: "My tool description",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
	}

	data, err := json.Marshal(td)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded ToolDefinition
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.Type != "function" {
		t.Errorf("Type = %q, want %q", decoded.Type, "function")
	}
	if decoded.Function.Name != "my_tool" {
		t.Errorf("Function.Name = %q, want %q", decoded.Function.Name, "my_tool")
	}
	if decoded.Function.Description != "My tool description" {
		t.Errorf("Function.Description = %q, want %q", decoded.Function.Description, "My tool description")
	}
}

func TestExecRequestDefaults(t *testing.T) {
	req := ExecRequest{}
	if req.Action != "" {
		t.Errorf("Action = %q, want empty", req.Action)
	}
	if req.Command != "" {
		t.Errorf("Command = %q, want empty", req.Command)
	}
	if req.Timeout != 0 {
		t.Errorf("Timeout = %d, want 0", req.Timeout)
	}
}

func TestExecRequestJSON(t *testing.T) {
	req := ExecRequest{
		Action:     "run",
		Command:    "echo hello",
		Timeout:    30,
		Background: true,
		Env:        map[string]string{"KEY": "val"},
		Cwd:        "/tmp",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded ExecRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.Action != "run" {
		t.Errorf("Action = %q, want %q", decoded.Action, "run")
	}
	if decoded.Command != "echo hello" {
		t.Errorf("Command = %q, want %q", decoded.Command, "echo hello")
	}
	if decoded.Timeout != 30 {
		t.Errorf("Timeout = %d, want 30", decoded.Timeout)
	}
	if !decoded.Background {
		t.Error("Background should be true")
	}
	if decoded.Env["KEY"] != "val" {
		t.Errorf("Env[\"KEY\"] = %q, want %q", decoded.Env["KEY"], "val")
	}
	if decoded.Cwd != "/tmp" {
		t.Errorf("Cwd = %q, want %q", decoded.Cwd, "/tmp")
	}
}

func TestExecResponseDefaults(t *testing.T) {
	resp := ExecResponse{}
	if resp.Status != "" {
		t.Errorf("Status = %q, want empty", resp.Status)
	}
	if resp.Output != "" {
		t.Errorf("Output = %q, want empty", resp.Output)
	}
	if resp.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", resp.ExitCode)
	}
}

func TestExecResponseJSON(t *testing.T) {
	resp := ExecResponse{
		SessionID: "sess_1",
		Status:    "running",
		ExitCode:  0,
		Output:    "hello world",
		Sessions: []SessionInfo{
			{
				ID:        "sess_1",
				Command:   "echo hello",
				Status:    "running",
				PID:       12345,
				StartedAt: 1000000,
			},
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded ExecResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.SessionID != "sess_1" {
		t.Errorf("SessionID = %q, want %q", decoded.SessionID, "sess_1")
	}
	if decoded.Status != "running" {
		t.Errorf("Status = %q, want %q", decoded.Status, "running")
	}
	if decoded.Output != "hello world" {
		t.Errorf("Output = %q, want %q", decoded.Output, "hello world")
	}
	if len(decoded.Sessions) != 1 {
		t.Fatalf("len(Sessions) = %d, want 1", len(decoded.Sessions))
	}
	if decoded.Sessions[0].ID != "sess_1" {
		t.Errorf("Sessions[0].ID = %q, want %q", decoded.Sessions[0].ID, "sess_1")
	}
	if decoded.Sessions[0].PID != 12345 {
		t.Errorf("Sessions[0].PID = %d, want 12345", decoded.Sessions[0].PID)
	}
}

func TestSessionInfoJSON(t *testing.T) {
	si := SessionInfo{
		ID:        "sess_1",
		Command:   "ls -la",
		Status:    "completed",
		PID:       54321,
		StartedAt: 2000000,
	}

	data, err := json.Marshal(si)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded SessionInfo
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.ID != "sess_1" {
		t.Errorf("ID = %q, want %q", decoded.ID, "sess_1")
	}
	if decoded.Command != "ls -la" {
		t.Errorf("Command = %q, want %q", decoded.Command, "ls -la")
	}
	if decoded.Status != "completed" {
		t.Errorf("Status = %q, want %q", decoded.Status, "completed")
	}
	if decoded.PID != 54321 {
		t.Errorf("PID = %d, want 54321", decoded.PID)
	}
	if decoded.StartedAt != 2000000 {
		t.Errorf("StartedAt = %d, want 2000000", decoded.StartedAt)
	}
}

func TestLLMProviderInterface(t *testing.T) {
	// Compile-time check that the interface is importable.
	var _ LLMProvider = &mockProvider{}
}

type mockProvider struct{}

func (m *mockProvider) Chat(_ context.Context, _ []Message, _ []ToolDefinition, _ string, _ map[string]any) (*LLMResponse, error) {
	return &LLMResponse{Content: "mock response", FinishReason: "stop"}, nil
}

func (m *mockProvider) GetDefaultModel() string {
	return "mock-model"
}
