package messageutil

import (
	"testing"

	"github.com/conglinyizhi/SylastraClaws/pkg/providers/protocoltypes"
)

// Helper to create messages compactly.
func msg(role, content, reasoning string, toolCalls, media, attachments int, toolCallID string) protocoltypes.Message {
	m := protocoltypes.Message{
		Role:             role,
		Content:          content,
		ReasoningContent: reasoning,
		ToolCallID:       toolCallID,
	}
	if toolCalls > 0 {
		m.ToolCalls = make([]protocoltypes.ToolCall, toolCalls)
	}
	if media > 0 {
		m.Media = make([]string, media)
	}
	if attachments > 0 {
		m.Attachments = make([]protocoltypes.Attachment, attachments)
	}
	return m
}

func TestIsTransientAssistantThoughtMessage(t *testing.T) {
	tests := []struct {
		name string
		msg  protocoltypes.Message
		want bool
	}{
		{
			name: "empty assistant thought message",
			msg: protocoltypes.Message{
				Role:             "assistant",
				Content:          "",
				ReasoningContent: "thinking...",
			},
			want: true,
		},
		{
			name: "assistant thought with whitespace content",
			msg: protocoltypes.Message{
				Role:             "assistant",
				Content:          "  \t  ",
				ReasoningContent: "thinking...",
			},
			want: true,
		},
		{
			name: "not assistant role",
			msg: protocoltypes.Message{
				Role:             "user",
				Content:          "",
				ReasoningContent: "thinking...",
			},
			want: false,
		},
		{
			name: "assistant with actual content",
			msg: protocoltypes.Message{
				Role:             "assistant",
				Content:          "Hello!",
				ReasoningContent: "thinking...",
			},
			want: false,
		},
		{
			name: "assistant with no reasoning content",
			msg: protocoltypes.Message{
				Role:    "assistant",
				Content: "",
			},
			want: false,
		},
		{
			name: "assistant with whitespace-only reasoning",
			msg: protocoltypes.Message{
				Role:             "assistant",
				Content:          "",
				ReasoningContent: "  ",
			},
			want: false,
		},
		{
			name: "assistant with tool calls",
			msg: protocoltypes.Message{
				Role:             "assistant",
				Content:          "",
				ReasoningContent: "thinking...",
				ToolCalls:        []protocoltypes.ToolCall{{ID: "call_1"}},
			},
			want: false,
		},
		{
			name: "assistant with media",
			msg: protocoltypes.Message{
				Role:             "assistant",
				Content:          "",
				ReasoningContent: "thinking...",
				Media:            []string{"img1"},
			},
			want: false,
		},
		{
			name: "assistant with attachments",
			msg: protocoltypes.Message{
				Role:             "assistant",
				Content:          "",
				ReasoningContent: "thinking...",
				Attachments:      []protocoltypes.Attachment{{Ref: "file1"}},
			},
			want: false,
		},
		{
			name: "assistant with tool call ID",
			msg: protocoltypes.Message{
				Role:             "assistant",
				Content:          "",
				ReasoningContent: "thinking...",
				ToolCallID:       "tcall_1",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsTransientAssistantThoughtMessage(tt.msg)
			if got != tt.want {
				t.Errorf("IsTransientAssistantThoughtMessage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterInvalidHistoryMessages(t *testing.T) {
	t.Run("nil input returns empty slice", func(t *testing.T) {
		result := FilterInvalidHistoryMessages(nil)
		if result == nil {
			t.Error("expected non-nil empty slice, got nil")
		}
		if len(result) != 0 {
			t.Errorf("expected empty slice, got %d elements", len(result))
		}
	})

	t.Run("empty input returns empty slice", func(t *testing.T) {
		result := FilterInvalidHistoryMessages([]protocoltypes.Message{})
		if result == nil {
			t.Error("expected non-nil empty slice, got nil")
		}
		if len(result) != 0 {
			t.Errorf("expected empty slice, got %d elements", len(result))
		}
	})

	t.Run("removes transient assistant thought messages", func(t *testing.T) {
		history := []protocoltypes.Message{
			{Role: "user", Content: "hello"},
			{Role: "assistant", Content: "", ReasoningContent: "hmm..."},
			{Role: "assistant", Content: "Hello! How can I help?"},
			{Role: "user", Content: "what is the weather?"},
		}
		result := FilterInvalidHistoryMessages(history)
		if len(result) != 3 {
			t.Fatalf("expected 3 messages, got %d", len(result))
		}
		// Verify transient thought message was removed (index 1 in original)
		if result[0].Role != "user" || result[0].Content != "hello" {
			t.Errorf("expected first message to be 'hello', got %+v", result[0])
		}
		if result[1].Role != "assistant" || result[1].Content != "Hello! How can I help?" {
			t.Errorf("expected second message to be assistant reply, got %+v", result[1])
		}
		if result[2].Role != "user" || result[2].Content != "what is the weather?" {
			t.Errorf("expected third message to be weather query, got %+v", result[2])
		}
	})

	t.Run("no transient messages returns same count", func(t *testing.T) {
		history := []protocoltypes.Message{
			{Role: "user", Content: "hi"},
			{Role: "assistant", Content: "hey there!"},
		}
		result := FilterInvalidHistoryMessages(history)
		if len(result) != 2 {
			t.Fatalf("expected 2 messages, got %d", len(result))
		}
	})

	t.Run("all transient messages returns empty", func(t *testing.T) {
		history := []protocoltypes.Message{
			{Role: "assistant", Content: "", ReasoningContent: "thought 1"},
			{Role: "assistant", Content: "", ReasoningContent: "thought 2"},
		}
		result := FilterInvalidHistoryMessages(history)
		if len(result) != 0 {
			t.Errorf("expected empty slice, got %d elements", len(result))
		}
	})
}
