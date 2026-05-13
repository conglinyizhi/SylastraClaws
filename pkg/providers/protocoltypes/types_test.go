package protocoltypes

import (
	"encoding/json"
	"testing"
)

func TestToolCallJSONRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		in   ToolCall
	}{
		{
			name: "basic tool call",
			in: ToolCall{
				ID:       "call_abc123",
				Type:     "function",
				Function: &FunctionCall{Name: "get_weather", Arguments: `{"location":"Beijing"}`},
			},
		},
		{
			name: "tool call with extra content",
			in: ToolCall{
				ID:               "call_xyz",
				Type:             "function",
				Function:         &FunctionCall{Name: "search", Arguments: `{"q":"test"}`},
				ExtraContent:     &ExtraContent{ToolFeedbackExplanation: "done"},
			},
		},
		{
			name: "tool call with omitempty fields empty",
			in: ToolCall{
				ID: "simple",
			},
		},
		{
			name: "tool call with google extra",
			in: ToolCall{
				ID:               "call_456",
				ExtraContent:     &ExtraContent{Google: &GoogleExtra{ThoughtSignature: "sig_1"}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.in)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}
			var out ToolCall
			if err := json.Unmarshal(data, &out); err != nil {
				t.Fatalf("Unmarshal error: %v (json: %s)", err, string(data))
			}
			if out.ID != tt.in.ID {
				t.Errorf("ID: got %q, want %q", out.ID, tt.in.ID)
			}
			if tt.in.Function != nil {
				if out.Function == nil {
					t.Fatal("Function is nil after unmarshal")
				}
				if out.Function.Name != tt.in.Function.Name {
					t.Errorf("Function.Name: got %q, want %q", out.Function.Name, tt.in.Function.Name)
				}
				if out.Function.Arguments != tt.in.Function.Arguments {
					t.Errorf("Function.Arguments: got %q, want %q", out.Function.Arguments, tt.in.Function.Arguments)
				}
			}
			if tt.in.ExtraContent != nil {
				if out.ExtraContent == nil {
					t.Fatal("ExtraContent is nil after unmarshal")
				}
				if out.ExtraContent.ToolFeedbackExplanation != tt.in.ExtraContent.ToolFeedbackExplanation {
					t.Errorf("ExtraContent.ToolFeedbackExplanation: got %q, want %q",
						out.ExtraContent.ToolFeedbackExplanation, tt.in.ExtraContent.ToolFeedbackExplanation)
				}
			}
			// Fields tagged with `json:"-"` should be empty after unmarshal
			if out.Arguments != nil {
				t.Error("Arguments should be empty (json:\"-\")")
			}
			if out.ThoughtSignature != "" {
				t.Error("ThoughtSignature should be empty (json:\"-\")")
			}
		})
	}
}

func TestFunctionCallJSONRoundTrip(t *testing.T) {
	fc := FunctionCall{
		Name:             "get_weather",
		Arguments:        `{"location":"Shanghai"}`,
		ThoughtSignature: "internal_sig",
	}
	data, err := json.Marshal(fc)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	var out FunctionCall
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if out.Name != fc.Name {
		t.Errorf("Name: got %q, want %q", out.Name, fc.Name)
	}
	if out.Arguments != fc.Arguments {
		t.Errorf("Arguments: got %q, want %q", out.Arguments, fc.Arguments)
	}
	if out.ThoughtSignature != fc.ThoughtSignature {
		t.Errorf("ThoughtSignature: got %q, want %q", out.ThoughtSignature, fc.ThoughtSignature)
	}
}

func TestFunctionCallOmitEmpty(t *testing.T) {
	fc := FunctionCall{
		Name:      "test",
		Arguments: "{}",
	}
	data, err := json.Marshal(fc)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	// ThoughtSignature should be omitted (omitempty)
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if _, exists := raw["thought_signature"]; exists {
		t.Error("thought_signature should be omitted when empty")
	}
}

func TestLLMResponseJSONRoundTrip(t *testing.T) {
	resp := LLMResponse{
		Content:          "Hello!",
		ReasoningContent: "thinking...",
		ToolCalls: []ToolCall{
			{ID: "call_1", Type: "function", Function: &FunctionCall{Name: "tool1", Arguments: "{}"}},
		},
		FinishReason: "stop",
		Usage:        &UsageInfo{PromptTokens: 10, CompletionTokens: 20, TotalTokens: 30},
		Reasoning:    "step by step",
		ReasoningDetails: []ReasoningDetail{
			{Format: "markdown", Index: 0, Type: "thinking", Text: "hmm"},
		},
	}
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	var out LLMResponse
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("Unmarshal error: %v (json: %s)", err, string(data))
	}
	if out.Content != resp.Content {
		t.Errorf("Content: got %q, want %q", out.Content, resp.Content)
	}
	if out.ReasoningContent != resp.ReasoningContent {
		t.Errorf("ReasoningContent: got %q, want %q", out.ReasoningContent, resp.ReasoningContent)
	}
	if out.FinishReason != resp.FinishReason {
		t.Errorf("FinishReason: got %q, want %q", out.FinishReason, resp.FinishReason)
	}
	if len(out.ToolCalls) != len(resp.ToolCalls) {
		t.Fatalf("ToolCalls length: got %d, want %d", len(out.ToolCalls), len(resp.ToolCalls))
	}
	if out.ToolCalls[0].ID != resp.ToolCalls[0].ID {
		t.Errorf("ToolCalls[0].ID: got %q, want %q", out.ToolCalls[0].ID, resp.ToolCalls[0].ID)
	}
	if out.ToolCalls[0].Function == nil || out.ToolCalls[0].Function.Name != resp.ToolCalls[0].Function.Name {
		t.Errorf("ToolCalls[0].Function.Name: got wrong value")
	}
	if out.Usage == nil {
		t.Fatal("Usage is nil")
	}
	if out.Usage.PromptTokens != 10 || out.Usage.CompletionTokens != 20 || out.Usage.TotalTokens != 30 {
		t.Errorf("Usage mismatch: %+v", out.Usage)
	}
	if len(out.ReasoningDetails) != 1 {
		t.Fatalf("ReasoningDetails length: got %d, want 1", len(out.ReasoningDetails))
	}
	if out.ReasoningDetails[0].Text != "hmm" {
		t.Errorf("ReasoningDetails[0].Text: got %q, want %q", out.ReasoningDetails[0].Text, "hmm")
	}
}

func TestLLMResponseEmptyToolCalls(t *testing.T) {
	resp := LLMResponse{
		Content:      "No tools",
		FinishReason: "stop",
	}
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if _, exists := raw["tool_calls"]; exists {
		t.Error("tool_calls should be omitted when empty")
	}
	if _, exists := raw["usage"]; exists {
		t.Error("usage should be omitted when empty")
	}
}

func TestMessageJSONRoundTrip(t *testing.T) {
	msg := Message{
		Role:             "assistant",
		Content:          "Hello!",
		Media:            []string{"img1", "img2"},
		Attachments:      []Attachment{{Type: "image", URL: "https://example.com/img.png", ContentType: "image/png"}},
		ReasoningContent: "thinking...",
		SystemParts:      []ContentBlock{{Type: "text", Text: "system instruction", CacheControl: &CacheControl{Type: "ephemeral"}}},
		ToolCalls:        []ToolCall{{ID: "call_1"}},
		ToolCallID:       "tcall_1",
	}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	var out Message
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("Unmarshal error: %v (json: %s)", err, string(data))
	}
	if out.Role != msg.Role {
		t.Errorf("Role: got %q, want %q", out.Role, msg.Role)
	}
	if out.Content != msg.Content {
		t.Errorf("Content: got %q, want %q", out.Content, msg.Content)
	}
	if len(out.Media) != 2 {
		t.Errorf("Media length: got %d, want 2", len(out.Media))
	}
	if len(out.SystemParts) != 1 {
		t.Fatalf("SystemParts length: got %d, want 1", len(out.SystemParts))
	}
	if out.SystemParts[0].CacheControl == nil || out.SystemParts[0].CacheControl.Type != "ephemeral" {
		t.Error("SystemParts[0].CacheControl missing or wrong")
	}
	if len(out.Attachments) != 1 {
		t.Fatalf("Attachments length: got %d, want 1", len(out.Attachments))
	}
	if out.Attachments[0].URL != "https://example.com/img.png" {
		t.Errorf("Attachment URL: got %q", out.Attachments[0].URL)
	}
}

func TestMessagePrivateFieldsNotSerialized(t *testing.T) {
	msg := Message{
		Role:         "user",
		Content:      "hi",
		PromptLayer:  "layer1",
		PromptSlot:   "slot1",
		PromptSource: "source1",
	}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if _, exists := raw["prompt_layer"]; exists {
		t.Error("prompt_layer should not be serialized (json:\"-\")")
	}
	if _, exists := raw["prompt_slot"]; exists {
		t.Error("prompt_slot should not be serialized (json:\"-\")")
	}
	if _, exists := raw["prompt_source"]; exists {
		t.Error("prompt_source should not be serialized (json:\"-\")")
	}
}

func TestUsageInfoRoundTrip(t *testing.T) {
	u := UsageInfo{PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150}
	data, err := json.Marshal(u)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	var out UsageInfo
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if out.PromptTokens != 100 || out.CompletionTokens != 50 || out.TotalTokens != 150 {
		t.Errorf("UsageInfo mismatch: %+v", out)
	}
}

func TestToolDefinitionRoundTrip(t *testing.T) {
	td := ToolDefinition{
		Type: "function",
		Function: ToolFunctionDefinition{
			Name:        "get_weather",
			Description: "Get current weather",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"location": map[string]any{"type": "string"},
				},
			},
		},
	}
	data, err := json.Marshal(td)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	var out ToolDefinition
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("Unmarshal error: %v (json: %s)", err, string(data))
	}
	if out.Type != "function" {
		t.Errorf("Type: got %q", out.Type)
	}
	if out.Function.Name != "get_weather" {
		t.Errorf("Function.Name: got %q", out.Function.Name)
	}
	if out.Function.Description != "Get current weather" {
		t.Errorf("Function.Description: got %q", out.Function.Description)
	}
	params, ok := out.Function.Parameters["properties"].(map[string]any)
	if !ok {
		t.Fatal("Parameters.properties is not map[string]any")
	}
	loc, ok := params["location"].(map[string]any)
	if !ok {
		t.Fatal("params['location'] is not map[string]any")
	}
	if loc["type"] != "string" {
		t.Errorf("location type: got %v", loc["type"])
	}
}

func TestExtraContentRoundTrip(t *testing.T) {
	ec := ExtraContent{
		Google:                  &GoogleExtra{ThoughtSignature: "sig_abc"},
		ToolFeedbackExplanation: "completed successfully",
	}
	data, err := json.Marshal(ec)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	var out ExtraContent
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("Unmarshal error: %v (json: %s)", err, string(data))
	}
	if out.ToolFeedbackExplanation != ec.ToolFeedbackExplanation {
		t.Errorf("ToolFeedbackExplanation: got %q, want %q", out.ToolFeedbackExplanation, ec.ToolFeedbackExplanation)
	}
	if out.Google == nil {
		t.Fatal("Google is nil")
	}
	if out.Google.ThoughtSignature != ec.Google.ThoughtSignature {
		t.Errorf("Google.ThoughtSignature: got %q, want %q", out.Google.ThoughtSignature, ec.Google.ThoughtSignature)
	}
}

func TestCacheControlRoundTrip(t *testing.T) {
	cc := CacheControl{Type: "ephemeral"}
	data, err := json.Marshal(cc)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	var out CacheControl
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if out.Type != "ephemeral" {
		t.Errorf("Type: got %q, want %q", out.Type, "ephemeral")
	}
}

func TestAttachmentJSONRoundTrip(t *testing.T) {
	a := Attachment{
		Type:        "image",
		Ref:         "media://abc",
		URL:         "https://example.com/img.png",
		Filename:    "photo.png",
		ContentType: "image/png",
	}
	data, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	var out Attachment
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if out.Type != a.Type || out.Ref != a.Ref || out.URL != a.URL || out.Filename != a.Filename || out.ContentType != a.ContentType {
		t.Errorf("Attachment mismatch: got %+v, want %+v", out, a)
	}
}

func TestContentBlockRoundTrip(t *testing.T) {
	cb := ContentBlock{
		Type: "text",
		Text: "Hello world",
	}
	data, err := json.Marshal(cb)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	var out ContentBlock
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("Unmarshal error: %v (json: %s)", err, string(data))
	}
	if out.Type != "text" || out.Text != "Hello world" {
		t.Errorf("ContentBlock mismatch: got %+v", out)
	}
}
