package toolshared

import (
	"bytes"
	"strings"
	"testing"
)

func TestDiffResult_UserVisibleUnifiedDiff(t *testing.T) {
	result := DiffResult("/tmp/example.txt", []byte("alpha\nbeta\ngamma\n"), []byte("alpha\nbeta 2\ngamma\n"))

	if result == nil {
		t.Fatal("DiffResult() returned nil")
	}
	if result.Silent {
		t.Fatal("expected DiffResult to be user-visible")
	}
	if result.IsError {
		t.Fatal("expected DiffResult to be successful")
	}

	if !strings.Contains(result.ForUser, "```diff") {
		t.Fatal("expected diff code block in user output")
	}
	if !strings.Contains(result.ForLLM, "File edited:") {
		t.Fatal("expected summary in LLM output")
	}
}

func TestDiffResult_NoContentChange(t *testing.T) {
	result := DiffResult("/tmp/same.txt", []byte("hello\n"), []byte("hello\n"))

	if result == nil {
		t.Fatal("DiffResult() returned nil")
	}
	if !strings.Contains(result.ForUser, "(no content change)") {
		t.Fatalf("expected 'no content change' in output, got: %s", result.ForUser)
	}
	if !strings.Contains(result.ForLLM, "(no content change)") {
		t.Fatalf("expected 'no content change' in LLM output, got: %s", result.ForLLM)
	}
}

func TestDiffResult_EmptyBefore(t *testing.T) {
	result := DiffResult("/tmp/new.txt", nil, []byte("hello\nworld\n"))

	if result == nil {
		t.Fatal("DiffResult() returned nil")
	}
	if result.IsError {
		t.Fatal("expected success for new file")
	}
	if !strings.Contains(result.ForUser, "+hello") {
		t.Fatalf("expected added content in diff, got: %s", result.ForUser)
	}
}

func TestDiffResult_EmptyAfter(t *testing.T) {
	result := DiffResult("/tmp/empty.txt", []byte("hello\nworld\n"), nil)

	if result == nil {
		t.Fatal("DiffResult() returned nil")
	}
	if result.IsError {
		t.Fatal("expected success for empty result")
	}
	if !strings.Contains(result.ForUser, "-hello") {
		t.Fatalf("expected removed content in diff, got: %s", result.ForUser)
	}
}

func TestDiffResult_BinaryContent(t *testing.T) {
	before := bytes.Repeat([]byte("a\n"), 100)
	after := bytes.Repeat([]byte("b\n"), 100)
	result := DiffResult("/tmp/binary.txt", before, after)

	if result == nil {
		t.Fatal("DiffResult() returned nil")
	}
	if result.IsError {
		t.Fatal("expected success for binary-like content")
	}
	if !strings.Contains(result.ForUser, "```diff") {
		t.Fatal("expected diff code block")
	}
}

func TestDiffResult_NilBeforeAfter(t *testing.T) {
	result := DiffResult("/tmp/nil.txt", nil, nil)

	if result == nil {
		t.Fatal("DiffResult() returned nil")
	}
	if result.IsError {
		t.Fatal("expected success for nil/nil")
	}
	if !strings.Contains(result.ForUser, "(no content change)") {
		t.Fatalf("expected 'no content change', got: %s", result.ForUser)
	}
}

func TestDiffResult_BeforeOnlyNewline(t *testing.T) {
	result := DiffResult("/tmp/newline.txt", []byte("\n"), []byte("hello\n"))

	if result == nil {
		t.Fatal("DiffResult() returned nil")
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	if !strings.Contains(result.ForUser, "+hello") {
		t.Fatalf("expected added content, got: %s", result.ForUser)
	}
}

func TestDiffResult_MissingTrailingNewline(t *testing.T) {
	result := DiffResult("/tmp/missing.txt", []byte("line1\nline2"), []byte("line1\nline2\nline3\n"))

	if result == nil {
		t.Fatal("DiffResult() returned nil")
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	if !strings.Contains(result.ForUser, `\ No newline at end of file`) {
		t.Fatalf("expected no-newline marker, got: %s", result.ForUser)
	}
}

func TestDiffResult_EscapedNewline(t *testing.T) {
	result := DiffResult("/tmp/escape.txt", []byte("\\n\n"), []byte("\\n literal\n"))

	if result == nil {
		t.Fatal("DiffResult() returned nil")
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	if !strings.Contains(result.ForUser, "-\\n") {
		t.Fatalf("expected removed backslash-n in diff, got: %s", result.ForUser)
	}
}
