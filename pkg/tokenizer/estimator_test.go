package tokenizer

import (
	"encoding/json"
	"testing"

	"github.com/conglinyizhi/SylastraClaws/pkg/providers"
)

// helper: estimateMessageChars computes the char count that
// EstimateMessageTokens uses before applying the 2/5 ratio and media tokens.
func estimateMessageChars(msg providers.Message) int {
	contentChars := runeCount(msg.Content)

	systemPartsChars := 0
	if len(msg.SystemParts) > 0 {
		for _, part := range msg.SystemParts {
			systemPartsChars += runeCount(part.Text)
		}
		const perPartOverhead = 20
		systemPartsChars += len(msg.SystemParts) * perPartOverhead
	}

	chars := contentChars
	if systemPartsChars > chars {
		chars = systemPartsChars
	}

	chars += runeCount(msg.ReasoningContent)

	for _, tc := range msg.ToolCalls {
		chars += len(tc.ID) + len(tc.Type)
		if tc.Function != nil {
			chars += len(tc.Function.Name) + len(tc.Function.Arguments)
		} else {
			chars += len(tc.Name)
		}
	}

	if msg.ToolCallID != "" {
		chars += len(msg.ToolCallID)
	}

	const messageOverhead = 12
	chars += messageOverhead

	return chars
}

func runeCount(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}

// --- EstimateMessageTokens tests ---

func TestEstimateMessageTokens_EmptyMessage(t *testing.T) {
	msg := providers.Message{}
	got := EstimateMessageTokens(msg)

	// chars = 0 (content) + 0 (reasoning) + 12 (overhead) = 12
	// tokens = 12 * 2 / 5 = 24 / 5 = 4
	want := 12 * 2 / 5
	if got != want {
		t.Errorf("EstimateMessageTokens(empty) = %d, want %d", got, want)
	}
}

func TestEstimateMessageTokens_OnlyContent(t *testing.T) {
	msg := providers.Message{Content: "Hello, world!"}
	got := EstimateMessageTokens(msg)

	chars := runeCount("Hello, world!") + 12 // 13 + 12 = 25
	want := chars * 2 / 5 // 50 / 5 = 10
	if got != want {
		t.Errorf("EstimateMessageTokens(content) = %d, want %d", got, want)
	}
}

func TestEstimateMessageTokens_OnlyReasoningContent(t *testing.T) {
	msg := providers.Message{
		Content:          "",
		ReasoningContent: "Let me think about this step by step...",
	}
	got := EstimateMessageTokens(msg)

	chars := 0 + runeCount("Let me think about this step by step...") + 12
	want := chars * 2 / 5
	if got != want {
		t.Errorf("EstimateMessageTokens(reasoning only) = %d, want %d", got, want)
	}
}

func TestEstimateMessageTokens_ContentAndReasoning(t *testing.T) {
	msg := providers.Message{
		Content:          "The answer is 42.",
		ReasoningContent: "I calculated 6 * 7.",
	}
	got := EstimateMessageTokens(msg)

	chars := runeCount("The answer is 42.") + runeCount("I calculated 6 * 7.") + 12
	want := chars * 2 / 5
	if got != want {
		t.Errorf("EstimateMessageTokens(content+reasoning) = %d, want %d", got, want)
	}
}

func TestEstimateMessageTokens_SystemPartsDominant(t *testing.T) {
	// SystemParts with more chars than Content → should use SystemParts as the base.
	msg := providers.Message{
		Content: "short",
		SystemParts: []providers.ContentBlock{
			{Text: "This is a very long system prompt that should dominate the character count"},
		},
	}
	got := EstimateMessageTokens(msg)

	systemChars := runeCount("This is a very long system prompt that should dominate the character count") + 20
	chars := systemChars + 0 + 12
	want := chars * 2 / 5
	if got != want {
		t.Errorf("EstimateMessageTokens(system dominating) = %d, want %d", got, want)
	}
}

func TestEstimateMessageTokens_ContentDominant(t *testing.T) {
	// Content with more chars than SystemParts → should use Content as the base.
	msg := providers.Message{
		Content: "This is a very long user message that should dominate the character count",
		SystemParts: []providers.ContentBlock{
			{Text: "short"},
		},
	}
	got := EstimateMessageTokens(msg)

	contentChars := runeCount("This is a very long user message that should dominate the character count")
	chars := contentChars + 0 + 12
	want := chars * 2 / 5
	if got != want {
		t.Errorf("EstimateMessageTokens(content dominating) = %d, want %d", got, want)
	}
}

func TestEstimateMessageTokens_MultipleSystemParts(t *testing.T) {
	msg := providers.Message{
		Content: "hello",
		SystemParts: []providers.ContentBlock{
			{Text: "part1"},
			{Text: "part2"},
			{Text: "part3"},
		},
	}
	got := EstimateMessageTokens(msg)

	systemChars := runeCount("part1") + runeCount("part2") + runeCount("part3") + 3*20
	chars := systemChars + 0 + 12
	want := chars * 2 / 5
	if got != want {
		t.Errorf("EstimateMessageTokens(multiple system parts) = %d, want %d", got, want)
	}
}

func TestEstimateMessageTokens_ToolCallsWithFunction(t *testing.T) {
	msg := providers.Message{
		Content: "What's the weather?",
		ToolCalls: []providers.ToolCall{
			{
				ID:   "call_123",
				Type: "function",
				Function: &providers.FunctionCall{
					Name:      "get_weather",
					Arguments: `{"location":"Beijing"}`,
				},
			},
		},
	}
	got := EstimateMessageTokens(msg)

	chars := runeCount("What's the weather?")
	chars += len("call_123") + len("function")
	chars += len("get_weather") + len(`{"location":"Beijing"}`)
	chars += 12
	want := chars * 2 / 5
	if got != want {
		t.Errorf("EstimateMessageTokens(tool call with function) = %d, want %d", got, want)
	}
}

func TestEstimateMessageTokens_ToolCallsWithoutFunction(t *testing.T) {
	msg := providers.Message{
		Content: "Search the web",
		ToolCalls: []providers.ToolCall{
			{
				ID:   "call_456",
				Type: "function",
				Name: "web_search", // top-level Name, no Function struct
			},
		},
	}
	got := EstimateMessageTokens(msg)

	chars := runeCount("Search the web")
	chars += len("call_456") + len("function")
	chars += len("web_search")
	chars += 12
	want := chars * 2 / 5
	if got != want {
		t.Errorf("EstimateMessageTokens(tool call without function) = %d, want %d", got, want)
	}
}

func TestEstimateMessageTokens_MultipleToolCalls(t *testing.T) {
	msg := providers.Message{
		Content: "Run all tasks",
		ToolCalls: []providers.ToolCall{
			{
				ID:   "c1",
				Type: "function",
				Function: &providers.FunctionCall{
					Name:      "task_a",
					Arguments: `{"x":1}`,
				},
			},
			{
				ID:   "c2",
				Type: "function",
				Function: &providers.FunctionCall{
					Name:      "task_b",
					Arguments: `{"y":2}`,
				},
			},
		},
	}
	got := EstimateMessageTokens(msg)

	chars := runeCount("Run all tasks")
	for _, tc := range msg.ToolCalls {
		chars += len(tc.ID) + len(tc.Type)
		if tc.Function != nil {
			chars += len(tc.Function.Name) + len(tc.Function.Arguments)
		}
	}
	chars += 12
	want := chars * 2 / 5
	if got != want {
		t.Errorf("EstimateMessageTokens(multiple tool calls) = %d, want %d", got, want)
	}
}

func TestEstimateMessageTokens_ToolCallID(t *testing.T) {
	msg := providers.Message{
		Content:    "Execute tool",
		ToolCallID: "prev_call_999",
	}
	got := EstimateMessageTokens(msg)

	chars := runeCount("Execute tool") + len("prev_call_999") + 12
	want := chars * 2 / 5
	if got != want {
		t.Errorf("EstimateMessageTokens(with ToolCallID) = %d, want %d", got, want)
	}
}

func TestEstimateMessageTokens_Media(t *testing.T) {
	msg := providers.Message{
		Content: "Check this image",
		Media:   []string{"https://example.com/image.png"},
	}
	got := EstimateMessageTokens(msg)

	chars := runeCount("Check this image") + 12
	baseTokens := chars * 2 / 5
	want := baseTokens + 256 // mediaTokensPerItem = 256
	if got != want {
		t.Errorf("EstimateMessageTokens(with 1 media) = %d, want %d", got, want)
	}
}

func TestEstimateMessageTokens_MultipleMedia(t *testing.T) {
	msg := providers.Message{
		Content: "Multiple images",
		Media:   []string{"img1.png", "img2.png", "img3.png"},
	}
	got := EstimateMessageTokens(msg)

	chars := runeCount("Multiple images") + 12
	baseTokens := chars * 2 / 5
	want := baseTokens + 3*256
	if got != want {
		t.Errorf("EstimateMessageTokens(with 3 media) = %d, want %d", got, want)
	}
}

func TestEstimateMessageTokens_AllDimensions(t *testing.T) {
	// Test a message that uses all fields simultaneously.
	msg := providers.Message{
		Content:          "Final answer.",
		ReasoningContent: "After careful thought...",
		ToolCalls: []providers.ToolCall{
			{
				ID:   "tc_1",
				Type: "function",
				Function: &providers.FunctionCall{
					Name:      "calc",
					Arguments: `{"expr":"2+2"}`,
				},
			},
		},
		Media: []string{"https://example.com/chart.png"},
	}
	got := EstimateMessageTokens(msg)

	chars := runeCount("Final answer.")
	chars += runeCount("After careful thought...")
	chars += len("tc_1") + len("function")
	chars += len("calc") + len(`{"expr":"2+2"}`)
	chars += 12
	baseTokens := chars * 2 / 5
	want := baseTokens + 256

	if got != want {
		t.Errorf("EstimateMessageTokens(all fields) = %d, want %d", got, want)
	}
}

func TestEstimateMessageTokens_UnicodeContent(t *testing.T) {
	// Unicode characters: RuneCountInString vs len matters.
	msg := providers.Message{
		Content: "你好世界！", // 5 CJK characters
	}
	got := EstimateMessageTokens(msg)

	// runeCount = 5, len("你好世界！") = 15 (UTF-8 bytes)
	// The code uses utf8.RuneCountInString, so it should count 5 chars
	chars := 5 + 12 // 17
	want := chars * 2 / 5

	if got != want {
		t.Errorf("EstimateMessageTokens(unicode) = %d, want %d", got, want)
	}
}

func TestEstimateMessageTokens_EmptySystemParts(t *testing.T) {
	// SystemParts slice is nil/empty → should not affect anything.
	msg := providers.Message{
		Content: "hello",
		SystemParts: []providers.ContentBlock{
			{Text: ""},
		},
	}
	got := EstimateMessageTokens(msg)

	// systemPartsChars = 0 (empty text) + 20 = 20
	// contentChars = 5 (rune count of "hello")
	// chars = max(5, 20) + 0 + 12 = 32
	chars := 20 + 0 + 12 // 32
	want := chars * 2 / 5
	if got != want {
		t.Errorf("EstimateMessageTokens(empty system part) = %d, want %d", got, want)
	}
}

// --- EstimateToolDefsTokens tests ---

func TestEstimateToolDefsTokens_Empty(t *testing.T) {
	got := EstimateToolDefsTokens(nil)
	if got != 0 {
		t.Errorf("EstimateToolDefsTokens(nil) = %d, want 0", got)
	}

	got = EstimateToolDefsTokens([]providers.ToolDefinition{})
	if got != 0 {
		t.Errorf("EstimateToolDefsTokens(empty) = %d, want 0", got)
	}
}

func TestEstimateToolDefsTokens_SingleTool(t *testing.T) {
	defs := []providers.ToolDefinition{
		{
			Type: "function",
			Function: providers.ToolFunctionDefinition{
				Name:        "get_weather",
				Description: "Get the current weather for a location",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"location": map[string]any{
							"type": "string",
						},
					},
				},
			},
		},
	}
	got := EstimateToolDefsTokens(defs)

	expectedChars := 0
	for _, d := range defs {
		expectedChars += len(d.Function.Name) + len(d.Function.Description)
		if d.Function.Parameters != nil {
			paramJSON, _ := json.Marshal(d.Function.Parameters)
			expectedChars += len(paramJSON)
		}
		expectedChars += 20
	}
	want := expectedChars * 2 / 5

	if got != want {
		t.Errorf("EstimateToolDefsTokens(single) = %d, want %d", got, want)
	}
}

func TestEstimateToolDefsTokens_MultipleTools(t *testing.T) {
	defs := []providers.ToolDefinition{
		{
			Type: "function",
			Function: providers.ToolFunctionDefinition{
				Name:        "tool_a",
				Description: "First tool",
			},
		},
		{
			Type: "function",
			Function: providers.ToolFunctionDefinition{
				Name:        "tool_b",
				Description: "Second tool with more description text",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"query": map[string]any{
							"type": "string",
						},
					},
				},
			},
		},
	}
	got := EstimateToolDefsTokens(defs)

	expectedChars := 0
	for _, d := range defs {
		expectedChars += len(d.Function.Name) + len(d.Function.Description)
		if d.Function.Parameters != nil {
			paramJSON, _ := json.Marshal(d.Function.Parameters)
			expectedChars += len(paramJSON)
		}
		expectedChars += 20
	}
	want := expectedChars * 2 / 5

	if got != want {
		t.Errorf("EstimateToolDefsTokens(multiple) = %d, want %d", got, want)
	}
}

func TestEstimateToolDefsTokens_NilParameters(t *testing.T) {
	defs := []providers.ToolDefinition{
		{
			Type: "function",
			Function: providers.ToolFunctionDefinition{
				Name:        "no_params",
				Description: "A tool with no parameters",
				Parameters:  nil,
			},
		},
	}
	got := EstimateToolDefsTokens(defs)

	expectedChars := len("no_params") + len("A tool with no parameters") + 20
	want := expectedChars * 2 / 5

	if got != want {
		t.Errorf("EstimateToolDefsTokens(nil params) = %d, want %d", got, want)
	}
}
