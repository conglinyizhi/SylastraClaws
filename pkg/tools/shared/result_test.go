package toolshared

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestNewToolResult(t *testing.T) {
	r := NewToolResult("hello world")
	if r.ForLLM != "hello world" {
		t.Errorf("ForLLM = %q, want %q", r.ForLLM, "hello world")
	}
	if r.ForUser != "" {
		t.Errorf("ForUser = %q, want empty", r.ForUser)
	}
	if r.Silent {
		t.Error("Silent should be false")
	}
	if r.IsError {
		t.Error("IsError should be false")
	}
	if r.Async {
		t.Error("Async should be false")
	}
	if r.ResponseHandled {
		t.Error("ResponseHandled should be false")
	}
}

func TestSilentResult(t *testing.T) {
	r := SilentResult("silent operation")
	if r.ForLLM != "silent operation" {
		t.Errorf("ForLLM = %q, want %q", r.ForLLM, "silent operation")
	}
	if !r.Silent {
		t.Error("Silent should be true")
	}
	if r.ForUser != "" {
		t.Errorf("ForUser should be empty for silent, got %q", r.ForUser)
	}
	if r.IsError {
		t.Error("IsError should be false")
	}
}

func TestErrorResult(t *testing.T) {
	r := ErrorResult("something broke")
	if r.ForLLM != "something broke" {
		t.Errorf("ForLLM = %q, want %q", r.ForLLM, "something broke")
	}
	if !r.IsError {
		t.Error("IsError should be true")
	}
	if r.Silent {
		t.Error("Silent should be false")
	}
}

func TestUserResult(t *testing.T) {
	r := UserResult("user visible content")
	if r.ForLLM != "user visible content" {
		t.Errorf("ForLLM = %q, want %q", r.ForLLM, "user visible content")
	}
	if r.ForUser != "user visible content" {
		t.Errorf("ForUser = %q, want %q", r.ForUser, "user visible content")
	}
	if r.Silent {
		t.Error("Silent should be false")
	}
	if r.IsError {
		t.Error("IsError should be false")
	}
}

func TestAsyncResult(t *testing.T) {
	r := AsyncResult("async operation started")
	if r.ForLLM != "async operation started" {
		t.Errorf("ForLLM = %q, want %q", r.ForLLM, "async operation started")
	}
	if !r.Async {
		t.Error("Async should be true")
	}
	if r.Silent {
		t.Error("Silent should be false")
	}
	if r.IsError {
		t.Error("IsError should be false")
	}
}

func TestMediaResult(t *testing.T) {
	mediaRefs := []string{"media://abc123", "media://def456"}
	r := MediaResult("image generated", mediaRefs)
	if r.ForLLM != "image generated" {
		t.Errorf("ForLLM = %q, want %q", r.ForLLM, "image generated")
	}
	if len(r.Media) != 2 {
		t.Fatalf("len(Media) = %d, want 2", len(r.Media))
	}
	if r.Media[0] != "media://abc123" {
		t.Errorf("Media[0] = %q, want %q", r.Media[0], "media://abc123")
	}
}

// --- ContentForLLM tests ---

func TestContentForLLM_Nil(t *testing.T) {
	var r *ToolResult
	if got := r.ContentForLLM(); got != "" {
		t.Errorf("ContentForLLM() on nil = %q, want empty", got)
	}
}

func TestContentForLLM_Empty(t *testing.T) {
	r := NewToolResult("")
	if got := r.ContentForLLM(); got != "" {
		t.Errorf("ContentForLLM() with empty = %q, want empty", got)
	}
}

func TestContentForLLM_ErrorFallback(t *testing.T) {
	err := errors.New("oops")
	r := ErrorResult("").WithError(err)
	if got := r.ContentForLLM(); got != "oops" {
		t.Errorf("ContentForLLM() = %q, want %q", got, "oops")
	}
}

func TestContentForLLM_ForLLMOverError(t *testing.T) {
	err := errors.New("hidden")
	r := ErrorResult("visible error").WithError(err)
	if got := r.ContentForLLM(); got != "visible error" {
		t.Errorf("ContentForLLM() = %q, want %q", got, "visible error")
	}
}

func TestContentForLLM_ResponseHandled(t *testing.T) {
	r := NewToolResult("")
	r.ResponseHandled = true
	got := r.ContentForLLM()
	if got != HandledToolLLMNote {
		t.Errorf("ContentForLLM() = %q, want %q", got, HandledToolLLMNote)
	}
}

func TestContentForLLM_ResponseHandledAppend(t *testing.T) {
	r := NewToolResult("some result")
	r.ResponseHandled = true
	got := r.ContentForLLM()
	if got != "some result\n"+HandledToolLLMNote {
		t.Errorf("ContentForLLM() = %q, want containing HandledToolLLMNote", got)
	}
}

func TestContentForLLM_ResponseHandledIdempotent(t *testing.T) {
	r := NewToolResult(HandledToolLLMNote)
	r.ResponseHandled = true
	got := r.ContentForLLM()
	// Should not append the note again.
	if got != HandledToolLLMNote {
		t.Errorf("ContentForLLM() = %q, want %q (no duplicate)", got, HandledToolLLMNote)
	}
}

func TestContentForLLM_ArtifactTags(t *testing.T) {
	r := NewToolResult("")
	r.ArtifactTags = []string{"[file:/tmp/test.png]"}
	got := r.ContentForLLM()
	want := "Local artifact paths: [file:/tmp/test.png]\n" + ArtifactPathsLLMNote
	if got != want {
		t.Errorf("ContentForLLM() = %q, want %q", got, want)
	}
}

func TestContentForLLM_ArtifactTagsAppend(t *testing.T) {
	r := NewToolResult("file saved")
	r.ArtifactTags = []string{"[file:/tmp/test.png]"}
	got := r.ContentForLLM()
	if !contains(got, ArtifactPathsLLMNote) {
		t.Errorf("ContentForLLM() should contain ArtifactPathsLLMNote, got %q", got)
	}
}

func TestContentForLLM_ResponseHandledAndArtifactTags(t *testing.T) {
	// When ResponseHandled=true and ForLLM is empty, ContentForLLM returns
	// only HandledToolLLMNote (short-circuits before ArtifactTags check).
	r := NewToolResult("")
	r.ResponseHandled = true
	r.ArtifactTags = []string{"[file:/tmp/test.png]"}
	got := r.ContentForLLM()
	if got != HandledToolLLMNote {
		t.Errorf("ContentForLLM() = %q, want %q (short-circuits on ResponseHandled with empty ForLLM)", got, HandledToolLLMNote)
	}
}

func TestContentForLLM_ResponseHandledAndArtifactTagsWithContent(t *testing.T) {
	// When ResponseHandled=true and ForLLM is non-empty, both notes appear.
	r := NewToolResult("some result")
	r.ResponseHandled = true
	r.ArtifactTags = []string{"[file:/tmp/test.png]"}
	got := r.ContentForLLM()
	if !contains(got, HandledToolLLMNote) {
		t.Errorf("ContentForLLM() should contain HandledToolLLMNote, got %q", got)
	}
	if !contains(got, "Local artifact paths") {
		t.Errorf("ContentForLLM() should contain artifact paths, got %q", got)
	}
}

// --- JSON serialization tests ---

func TestToolResultMarshalJSON_Basic(t *testing.T) {
	r := NewToolResult("llm content")
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}

	if result["for_llm"] != "llm content" {
		t.Errorf("for_llm = %q, want %q", result["for_llm"], "llm content")
	}
	if result["silent"] != false {
		t.Errorf("silent = %v, want false", result["silent"])
	}
	if result["is_error"] != false {
		t.Errorf("is_error = %v, want false", result["is_error"])
	}
	if result["async"] != false {
		t.Errorf("async = %v, want false", result["async"])
	}
}

func TestToolResultMarshalJSON_ForUser(t *testing.T) {
	r := UserResult("user message")
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}

	if result["for_user"] != "user message" {
		t.Errorf("for_user = %q, want %q", result["for_user"], "user message")
	}
}

func TestToolResultMarshalJSON_ForUserOmittedWhenEmpty(t *testing.T) {
	r := NewToolResult("only llm")
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}

	if _, ok := result["for_user"]; ok {
		t.Error("for_user should be omitted when empty (omitempty)")
	}
}

func TestToolResultMarshalJSON_ErrorFieldExcluded(t *testing.T) {
	r := ErrorResult("error message").WithError(errors.New("internal error"))
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}

	// Err field should not appear in JSON (json:"-")
	if _, ok := result["Err"]; ok {
		t.Error("Err field should not be in JSON output")
	}
	if _, ok := result["err"]; ok {
		t.Error("'err' key should not be in JSON output")
	}
}

func TestToolResultMarshalJSON_Media(t *testing.T) {
	r := MediaResult("media result", []string{"media://abc"})
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}

	media, ok := result["media"].([]any)
	if !ok {
		t.Fatal("media field should be an array")
	}
	if len(media) != 1 || media[0] != "media://abc" {
		t.Errorf("media = %v, want [media://abc]", media)
	}
}

func TestToolResultMarshalJSON_Silent(t *testing.T) {
	r := SilentResult("silent")
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}

	silent, ok := result["silent"]
	if !ok {
		t.Fatal("silent field should be present")
	}
	silentBool, ok := silent.(bool)
	if !ok || !silentBool {
		t.Errorf("silent = %v, want true", silent)
	}
}

func TestToolResultMarshalJSON_MessagesFieldExcluded(t *testing.T) {
	r := NewToolResult("test")
	r.Messages = nil // Messages has json:"-" so shouldn't appear
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}

	if _, ok := result["messages"]; ok {
		t.Error("'messages' key should not be in JSON output (json:\"-\")")
	}
}

func TestToolResultMarshalJSON_ArtifactTags(t *testing.T) {
	r := NewToolResult("with artifacts")
	r.ArtifactTags = []string{"[file:/tmp/a.png]", "[file:/tmp/b.txt]"}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}

	tags, ok := result["artifact_tags"].([]any)
	if !ok {
		t.Fatal("artifact_tags field should be an array")
	}
	if len(tags) != 2 {
		t.Fatalf("len(artifact_tags) = %d, want 2", len(tags))
	}
	if tags[0] != "[file:/tmp/a.png]" {
		t.Errorf("artifact_tags[0] = %q, want %q", tags[0], "[file:/tmp/a.png]")
	}
}

func TestToolResultMarshalJSON_IsError(t *testing.T) {
	r := ErrorResult("failure")
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}

	isErr, ok := result["is_error"]
	if !ok {
		t.Fatal("is_error field should be present")
	}
	isErrBool, ok := isErr.(bool)
	if !ok || !isErrBool {
		t.Errorf("is_error = %v, want true", isErr)
	}
}

// --- Unmarshal / round-trip tests ---

func TestToolResultUnmarshalJSON(t *testing.T) {
	input := `{
		"for_llm": "llm content",
		"for_user": "user content",
		"silent": true,
		"is_error": false,
		"async": true,
		"media": ["media://abc"],
		"artifact_tags": ["[file:/tmp/x.png]"]
	}`

	var r ToolResult
	if err := json.Unmarshal([]byte(input), &r); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}

	if r.ForLLM != "llm content" {
		t.Errorf("ForLLM = %q, want %q", r.ForLLM, "llm content")
	}
	if r.ForUser != "user content" {
		t.Errorf("ForUser = %q, want %q", r.ForUser, "user content")
	}
	if !r.Silent {
		t.Error("Silent should be true")
	}
	if r.IsError {
		t.Error("IsError should be false")
	}
	if !r.Async {
		t.Error("Async should be true")
	}
	if len(r.Media) != 1 || r.Media[0] != "media://abc" {
		t.Errorf("Media = %v, want [media://abc]", r.Media)
	}
	if len(r.ArtifactTags) != 1 || r.ArtifactTags[0] != "[file:/tmp/x.png]" {
		t.Errorf("ArtifactTags = %v, want [[file:/tmp/x.png]]", r.ArtifactTags)
	}
}

func TestToolResultJSONRoundTrip(t *testing.T) {
	original := &ToolResult{
		ForLLM:          "llm text",
		ForUser:         "user text",
		Silent:          true,
		IsError:         false,
		Async:           true,
		Err:             errors.New("internal only"),
		Media:           []string{"media://abc"},
		ArtifactTags:    []string{"[file:/tmp/a.png]"},
		ResponseHandled: true,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded ToolResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.ForLLM != original.ForLLM {
		t.Errorf("ForLLM = %q, want %q", decoded.ForLLM, original.ForLLM)
	}
	if decoded.ForUser != original.ForUser {
		t.Errorf("ForUser = %q, want %q", decoded.ForUser, original.ForUser)
	}
	if decoded.Silent != original.Silent {
		t.Errorf("Silent = %v, want %v", decoded.Silent, original.Silent)
	}
	if decoded.IsError != original.IsError {
		t.Errorf("IsError = %v, want %v", decoded.IsError, original.IsError)
	}
	if decoded.Async != original.Async {
		t.Errorf("Async = %v, want %v", decoded.Async, original.Async)
	}
	if decoded.ResponseHandled != original.ResponseHandled {
		t.Errorf("ResponseHandled = %v, want %v", decoded.ResponseHandled, original.ResponseHandled)
	}
	// Err should NOT survive round-trip (json:"-")
	if decoded.Err != nil {
		t.Error("Err should be nil after unmarshal (json:\"-\")")
	}
	// Media
	if len(decoded.Media) != 1 || decoded.Media[0] != "media://abc" {
		t.Errorf("Media = %v, want [media://abc]", decoded.Media)
	}
	// ArtifactTags
	if len(decoded.ArtifactTags) != 1 || decoded.ArtifactTags[0] != "[file:/tmp/a.png]" {
		t.Errorf("ArtifactTags = %v, want [[file:/tmp/a.png]]", decoded.ArtifactTags)
	}
	// Messages should be nil (json:"-")
	if decoded.Messages != nil {
		t.Error("Messages should be nil after unmarshal (json:\"-\")")
	}
}

// --- WithError chaining ---

func TestWithError(t *testing.T) {
	origErr := errors.New("original error")
	r := SilentResult("fail").WithError(origErr)
	if r.Err != origErr {
		t.Errorf("Err = %v, want %v", r.Err, origErr)
	}
	// Ensure chaining preserves Silent.
	if !r.Silent {
		t.Error("Silent should be preserved after WithError")
	}
}

func TestWithResponseHandled(t *testing.T) {
	r := NewToolResult("done").WithResponseHandled()
	if !r.ResponseHandled {
		t.Error("ResponseHandled should be true after WithResponseHandled()")
	}
}

func TestWithResponseHandledChaining(t *testing.T) {
	err := errors.New("oops")
	r := ErrorResult("error happened").
		WithError(err).
		WithResponseHandled()
	if !r.IsError {
		t.Error("IsError should be true")
	}
	if r.Err != err {
		t.Errorf("Err = %v, want %v", r.Err, err)
	}
	if !r.ResponseHandled {
		t.Error("ResponseHandled should be true")
	}
}

// --- Constant values ---

func TestHandledToolLLMNote(t *testing.T) {
	if HandledToolLLMNote == "" {
		t.Error("HandledToolLLMNote must not be empty")
	}
	if !contains(HandledToolLLMNote, "already been delivered") {
		t.Errorf("HandledToolLLMNote should mention delivery, got %q", HandledToolLLMNote)
	}
}

func TestArtifactPathsLLMNote(t *testing.T) {
	if ArtifactPathsLLMNote == "" {
		t.Error("ArtifactPathsLLMNote must not be empty")
	}
	if !contains(ArtifactPathsLLMNote, "send_file") {
		t.Errorf("ArtifactPathsLLMNote should mention send_file, got %q", ArtifactPathsLLMNote)
	}
}

// --- Helper ---

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsStr(s, substr)
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
