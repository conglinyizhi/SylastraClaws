package fstools

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	betools "github.com/conglinyizhi/better-edit-tools-mcp/pkg/betools"
)

// BetterReadTool implements the Tool interface for betools Show operation.
type BetterReadTool struct {
	workspace  string
	restrict   bool
	allowPaths []*regexp.Regexp
}

func NewBetterReadTool(workspace string, restrict bool, allowPaths ...[]*regexp.Regexp) *BetterReadTool {
	var patterns []*regexp.Regexp
	if len(allowPaths) > 0 {
		patterns = allowPaths[0]
	}
	return &BetterReadTool{
		workspace:  workspace,
		restrict:   restrict,
		allowPaths: patterns,
	}
}

func (t *BetterReadTool) Name() string {
	return "better_read"
}

func (t *BetterReadTool) Description() string {
	return "Read file content with line numbers using better-edit-tools. Supports range display (start/end) and auto-mode for detecting function scope."
}

func (t *BetterReadTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"file": map[string]any{
				"type":        "string",
				"description": "The file path to display.",
			},
			"start": map[string]any{
				"type":        "integer",
				"description": "Starting line number (1-indexed). Default: 1.",
			},
			"end": map[string]any{
				"type":        "integer",
				"description": "Ending line number. Use 0 (default) for auto-mode that detects function scope or shows context around start.",
			},
		},
		"required": []string{"file"},
	}
}

func (t *BetterReadTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	path, _ := args["file"].(string)
	if path == "" {
		return ErrorResult("file is required")
	}

	resolvedPath, err := validatePathWithAllowPaths(path, t.workspace, t.restrict, t.allowPaths)
	if err != nil {
		return ErrorResult(fmt.Sprintf("invalid path: %v", err))
	}

	start, _ := toInt(args["start"])
	if start < 1 {
		start = 1
	}
	end, _ := toInt(args["end"])
	if end < 0 {
		end = 0
	}

	result, sessionID, err := betools.Read(resolvedPath, start, end)
	if err != nil {
		return ErrorResult(fmt.Sprintf("better_read failed: %v", err))
	}

	output := fmt.Sprintf("File: %s (lines %d-%d of %d)\n", result.File, result.Start, result.End, result.Total)
	if sessionID != "" {
		output += fmt.Sprintf("Session: %s\n", sessionID)
	}
	output += "\n" + result.Content

	return NewToolResult(output)
}

// BetterReplaceTool implements the Tool interface for betools Replace operation.
type BetterReplaceTool struct {
	workspace  string
	restrict   bool
	allowPaths []*regexp.Regexp
}

func NewBetterReplaceTool(workspace string, restrict bool, allowPaths ...[]*regexp.Regexp) *BetterReplaceTool {
	var patterns []*regexp.Regexp
	if len(allowPaths) > 0 {
		patterns = allowPaths[0]
	}
	return &BetterReplaceTool{
		workspace:  workspace,
		restrict:   restrict,
		allowPaths: patterns,
	}
}

func (t *BetterReplaceTool) Name() string {
	return "better_replace"
}

func (t *BetterReplaceTool) Description() string {
	return "Replace a range of lines in a file. Requires start/end line numbers. Supports optional old content validation to prevent drift. Returns a diff of the change."
}

func (t *BetterReplaceTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"file": map[string]any{
				"type":        "string",
				"description": "The file path to edit.",
			},
			"start": map[string]any{
				"type":        "integer",
				"description": "Starting line number (1-indexed, inclusive) of the range to replace.",
			},
			"end": map[string]any{
				"type":        "integer",
				"description": "Ending line number (1-indexed, inclusive) of the range to replace.",
			},
			"old": map[string]any{
				"type":        "string",
				"description": "Optional. The exact content of the range being replaced, used to verify the file hasn't drifted before editing.",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "The new content to replace with. Standard JSON escaping: \\n for newline.",
			},
			"format": map[string]any{
				"type":        "string",
				"description": "Diff output format: 'unified' (default) or 'json'.",
				"default":     "unified",
			},
			"preview": map[string]any{
				"type":        "boolean",
				"description": "If true, only show the diff without applying changes.",
				"default":     false,
			},
		},
		"required": []string{"file", "start", "end", "content"},
	}
}

func (t *BetterReplaceTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	path, _ := args["file"].(string)
	if path == "" {
		return ErrorResult("file is required")
	}

	resolvedPath, err := validatePathWithAllowPaths(path, t.workspace, t.restrict, t.allowPaths)
	if err != nil {
		return ErrorResult(fmt.Sprintf("invalid path: %v", err))
	}

	start, ok := toInt(args["start"])
	if !ok || start < 1 {
		return ErrorResult("start is required and must be >= 1")
	}
	end, ok := toInt(args["end"])
	if !ok || end < start {
		return ErrorResult("end is required and must be >= start")
	}
	content, _ := args["content"].(string)

	var old *string
	if oldRaw, exists := args["old"]; exists && oldRaw != nil {
		if oldStr, ok := oldRaw.(string); ok && oldStr != "" {
			old = &oldStr
		}
	}
	format, _ := args["format"].(string)
	if format == "" {
		format = "unified"
	}
	preview, _ := args["preview"].(bool)

	result, err := betools.Replace(resolvedPath, start, end, old, content, false, format, preview, "")
	if err != nil {
		return ErrorResult(fmt.Sprintf("better_replace failed: %v", err))
	}

	output := fmt.Sprintf("File: %s | Status: %s\n", result.File, result.Status)
	output += fmt.Sprintf("Removed: %d lines, Added: %d lines, Total: %d lines\n", result.Removed, result.Added, result.Total)
	if result.Diff != "" {
		output += "\nDiff:\n" + result.Diff
	}
	if result.Balance != "" {
		output += "\nBracket Balance: " + result.Balance
	}
	if result.Warning != "" {
		output += "\nWarning: " + result.Warning
	}
	if preview {
		output += "\n(Preview only - changes were NOT applied)"
	}

	return NewToolResult(output)
}

// BetterInsertTool implements the Tool interface for betools Insert operation.
type BetterInsertTool struct {
	workspace  string
	restrict   bool
	allowPaths []*regexp.Regexp
}

func NewBetterInsertTool(workspace string, restrict bool, allowPaths ...[]*regexp.Regexp) *BetterInsertTool {
	var patterns []*regexp.Regexp
	if len(allowPaths) > 0 {
		patterns = allowPaths[0]
	}
	return &BetterInsertTool{
		workspace:  workspace,
		restrict:   restrict,
		allowPaths: patterns,
	}
}

func (t *BetterInsertTool) Name() string {
	return "better_insert"
}

func (t *BetterInsertTool) Description() string {
	return "Insert new content after a specific line number. Use after=0 to insert at the beginning of the file, or after=<last_line> to effectively append."
}

func (t *BetterInsertTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"file": map[string]any{
				"type":        "string",
				"description": "The file path to edit.",
			},
			"after": map[string]any{
				"type":        "integer",
				"description": "Insert new content after this line number. Use 0 to insert at the beginning.",
				"default":     0,
			},
			"content": map[string]any{
				"type":        "string",
				"description": "The content to insert. Standard JSON escaping: \\n for newline.",
			},
			"preview": map[string]any{
				"type":        "boolean",
				"description": "If true, only show the diff without applying changes.",
				"default":     false,
			},
		},
		"required": []string{"file", "content"},
	}
}

func (t *BetterInsertTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	path, _ := args["file"].(string)
	if path == "" {
		return ErrorResult("file is required")
	}

	resolvedPath, err := validatePathWithAllowPaths(path, t.workspace, t.restrict, t.allowPaths)
	if err != nil {
		return ErrorResult(fmt.Sprintf("invalid path: %v", err))
	}

	after, _ := toInt(args["after"])
	if after < 0 {
		after = 0
	}
	content, _ := args["content"].(string)
	if content == "" {
		return ErrorResult("content is required")
	}
	preview, _ := args["preview"].(bool)

	result, err := betools.Insert(resolvedPath, after, content, false, "unified", preview)
	if err != nil {
		return ErrorResult(fmt.Sprintf("better_insert failed: %v", err))
	}

	output := fmt.Sprintf("File: %s | Status: %s\n", result.File, result.Status)
	output += fmt.Sprintf("Inserted after line %d: %d lines added, Total: %d lines\n", result.After, result.Added, result.Total)
	if result.Diff != "" {
		output += "\nDiff:\n" + result.Diff
	}
	if result.Balance != "" {
		output += "\nBracket Balance: " + result.Balance
	}
	if preview {
		output += "\n(Preview only - changes were NOT applied)"
	}

	return NewToolResult(output)
}

// BetterDeleteTool implements the Tool interface for betools Delete operation.
type BetterDeleteTool struct {
	workspace  string
	restrict   bool
	allowPaths []*regexp.Regexp
}

func NewBetterDeleteTool(workspace string, restrict bool, allowPaths ...[]*regexp.Regexp) *BetterDeleteTool {
	var patterns []*regexp.Regexp
	if len(allowPaths) > 0 {
		patterns = allowPaths[0]
	}
	return &BetterDeleteTool{
		workspace:  workspace,
		restrict:   restrict,
		allowPaths: patterns,
	}
}

func (t *BetterDeleteTool) Name() string {
	return "better_delete"
}

func (t *BetterDeleteTool) Description() string {
	return "Delete a range of lines or specific line numbers from a file. Specify start/end for a range, or use lines as a JSON array for non-contiguous deletions."
}

func (t *BetterDeleteTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"file": map[string]any{
				"type":        "string",
				"description": "The file path to edit.",
			},
			"start": map[string]any{
				"type":        "integer",
				"description": "Starting line number (1-indexed, inclusive) of the range to delete.",
			},
			"end": map[string]any{
				"type":        "integer",
				"description": "Ending line number (1-indexed, inclusive). Default: same as start for single-line delete.",
			},
			"lines": map[string]any{
				"type":        "string",
				"description": "JSON array of line numbers for non-contiguous deletions, e.g. [3,5,7]. Overrides start/end if provided.",
			},
			"preview": map[string]any{
				"type":        "boolean",
				"description": "If true, only show the diff without applying changes.",
				"default":     false,
			},
		},
		"required": []string{"file"},
	}
}

func (t *BetterDeleteTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	path, _ := args["file"].(string)
	if path == "" {
		return ErrorResult("file is required")
	}

	resolvedPath, err := validatePathWithAllowPaths(path, t.workspace, t.restrict, t.allowPaths)
	if err != nil {
		return ErrorResult(fmt.Sprintf("invalid path: %v", err))
	}

	start, _ := toInt(args["start"])
	if start < 1 {
		start = 0
	}
	end, _ := toInt(args["end"])
	if end < 0 {
		end = 0
	}

	var linesJSON *string
	if linesRaw, exists := args["lines"]; exists && linesRaw != nil {
		if linesStr, ok := linesRaw.(string); ok && linesStr != "" {
			linesJSON = &linesStr
		}
	}
	preview, _ := args["preview"].(bool)

	result, err := betools.Delete(resolvedPath, start, end, 0, linesJSON, "unified", preview)
	if err != nil {
		return ErrorResult(fmt.Sprintf("better_delete failed: %v", err))
	}

	output := fmt.Sprintf("File: %s | Status: %s\n", result.File, result.Status)
	output += fmt.Sprintf("Total: %d lines\n", result.Total)
	if result.Diff != "" {
		output += "\nDiff:\n" + result.Diff
	}
	if result.Balance != "" {
		output += "\nBracket Balance: " + result.Balance
	}
	if preview {
		output += "\n(Preview only - changes were NOT applied)"
	}

	return NewToolResult(output)
}

// BetterBatchTool implements the Tool interface for betools Batch operation.
type BetterBatchTool struct {
	workspace  string
	restrict   bool
	allowPaths []*regexp.Regexp
}

func NewBetterBatchTool(workspace string, restrict bool, allowPaths ...[]*regexp.Regexp) *BetterBatchTool {
	var patterns []*regexp.Regexp
	if len(allowPaths) > 0 {
		patterns = allowPaths[0]
	}
	return &BetterBatchTool{
		workspace:  workspace,
		restrict:   restrict,
		allowPaths: patterns,
	}
}

func (t *BetterBatchTool) Name() string {
	return "better_batch"
}

func (t *BetterBatchTool) Description() string {
	return "Apply multiple edits across multiple files in a single atomic operation. The spec parameter accepts a JSON array or object describing edits per file."
}

func (t *BetterBatchTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"spec": map[string]any{
				"type":        "string",
				"description": "JSON spec describing edits. Supports single file or multi-file format. Each file entry has a 'file' path and 'edits' array with actions (replace-lines, insert-after, delete-lines).",
			},
			"preview": map[string]any{
				"type":        "boolean",
				"description": "If true, only validate without applying.",
				"default":     false,
			},
		},
		"required": []string{"spec"},
	}
}

func (t *BetterBatchTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	rawSpec, _ := args["spec"].(string)
	if rawSpec == "" {
		return ErrorResult("spec is required")
	}

	// Extract and validate file paths from the spec
	resolvedSpec, err := t.resolveBatchSpec(rawSpec)
	if err != nil {
		return ErrorResult(fmt.Sprintf("invalid spec: %v", err))
	}

	preview, _ := args["preview"].(bool)

	result, err := betools.Batch(resolvedSpec, preview)
	if err != nil {
		return ErrorResult(fmt.Sprintf("better_batch failed: %v", err))
	}

	var output string
	if len(result.Results) == 1 {
		r := result.Results[0]
		output = fmt.Sprintf("File: %s | Edits: %d | Total: %d lines\n", r.File, r.Edits, r.Total)
	} else {
		output = fmt.Sprintf("Files: %d\n", result.Files)
		for _, r := range result.Results {
			output += fmt.Sprintf("  - %s: %d edits, %d lines\n", r.File, r.Edits, r.Total)
		}
	}
	if preview {
		output += "(Preview only - changes were NOT applied)"
	}

	return NewToolResult(output)
}

// resolveBatchSpec walks the spec JSON to extract and validate file paths
func (t *BetterBatchTool) resolveBatchSpec(spec string) (string, error) {
	var raw any
	if err := json.Unmarshal([]byte(spec), &raw); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}

	// Walk the JSON to find all "file" fields and validate them
	resolved, err := walkAndValidatePaths(raw, t.workspace, t.restrict, t.allowPaths)
	if err != nil {
		return "", err
	}

	b, err := json.Marshal(resolved)
	if err != nil {
		return "", fmt.Errorf("failed to re-serialize spec: %w", err)
	}
	return string(b), nil
}

func walkAndValidatePaths(v any, workspace string, restrict bool, allowPaths []*regexp.Regexp) (any, error) {
	switch val := v.(type) {
	case map[string]any:
		resolved := make(map[string]any, len(val))
		for k, child := range val {
			if k == "file" {
				if path, ok := child.(string); ok {
					resolvedPath, err := validatePathWithAllowPaths(path, workspace, restrict, allowPaths)
					if err != nil {
						return nil, fmt.Errorf("path %q: %w", path, err)
					}
					resolved[k] = resolvedPath
					continue
				}
			}
			r, err := walkAndValidatePaths(child, workspace, restrict, allowPaths)
			if err != nil {
				return nil, err
			}
			resolved[k] = r
		}
		return resolved, nil
	case []any:
		resolved := make([]any, len(val))
		for i, child := range val {
			r, err := walkAndValidatePaths(child, workspace, restrict, allowPaths)
			if err != nil {
				return nil, err
			}
			resolved[i] = r
		}
		return resolved, nil
	default:
		return v, nil
	}
}

// BetterWriteTool implements the Tool interface for betools Write operation.
type BetterWriteTool struct {
	workspace  string
	restrict   bool
	allowPaths []*regexp.Regexp
}

func NewBetterWriteTool(workspace string, restrict bool, allowPaths ...[]*regexp.Regexp) *BetterWriteTool {
	var patterns []*regexp.Regexp
	if len(allowPaths) > 0 {
		patterns = allowPaths[0]
	}
	return &BetterWriteTool{
		workspace:  workspace,
		restrict:   restrict,
		allowPaths: patterns,
	}
}

func (t *BetterWriteTool) Name() string {
	return "better_write"
}

func (t *BetterWriteTool) Description() string {
	return "Create or overwrite files. Accepts a JSON spec with one or more file entries. Uses degraded JSON parser for malformed input. Writes are atomic (temp file + rename)."
}

func (t *BetterWriteTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"spec": map[string]any{
				"type":        "string",
				"description": "JSON spec: {file: path, content: text} or {files: [{file: path, content: text}, ...]}. Supports extract: true to extract content from code blocks.",
			},
			"preview": map[string]any{
				"type":        "boolean",
				"description": "If true, only validate paths without writing.",
				"default":     false,
			},
			"raw": map[string]any{
				"type":        "boolean",
				"description": "If true, skip JSON escape processing for content.",
				"default":     false,
			},
		},
		"required": []string{"spec"},
	}
}

func (t *BetterWriteTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	rawSpec, _ := args["spec"].(string)
	if rawSpec == "" {
		return ErrorResult("spec is required")
	}

	// Extract and validate file paths from the spec
	resolvedSpec, err := t.resolveWriteSpec(rawSpec)
	if err != nil {
		return ErrorResult(fmt.Sprintf("invalid spec: %v", err))
	}

	preview, _ := args["preview"].(bool)
	raw, _ := args["raw"].(bool)

	result, err := betools.Write(resolvedSpec, preview, raw)
	if err != nil {
		return ErrorResult(fmt.Sprintf("better_write failed: %v", err))
	}

	var output string
	if len(result.Results) == 1 {
		r := result.Results[0]
		output = fmt.Sprintf("File: %s | %d lines, %d bytes\n", r.File, r.Lines, r.Bytes)
	} else {
		output = fmt.Sprintf("Files written: %d\n", result.Files)
		for _, r := range result.Results {
			output += fmt.Sprintf("  - %s: %d lines, %d bytes\n", r.File, r.Lines, r.Bytes)
		}
	}
	if result.Degraded {
		output += "\nWarning: " + result.Warning
	}
	if preview {
		output += "(Preview only - changes were NOT applied)"
	}

	return NewToolResult(output)
}

// resolveWriteSpec parses the spec JSON, extracts file paths, validates them, and re-serializes.
func (t *BetterWriteTool) resolveWriteSpec(spec string) (string, error) {
	var raw any
	// Try standard JSON first
	if err := json.Unmarshal([]byte(spec), &raw); err != nil {
		// If standard JSON fails, try to extract file paths from raw text
		// This is a best-effort approach for degraded mode
		resolved, err := t.resolveDegradedPaths(spec)
		if err != nil {
			return "", fmt.Errorf("invalid spec and degraded parse failed: %w", err)
		}
		return resolved, nil
	}

	resolved, err := walkAndValidatePaths(raw, t.workspace, t.restrict, t.allowPaths)
	if err != nil {
		return "", err
	}

	b, err := json.Marshal(resolved)
	if err != nil {
		return "", fmt.Errorf("failed to re-serialize spec: %w", err)
	}
	return string(b), nil
}

// resolveDegradedPaths attempts to extract file paths from a malformed JSON spec string.
// This is a fallback for the degraded JSON parser in betools.Write.
func (t *BetterWriteTool) resolveDegradedPaths(spec string) (string, error) {
	// Try to extract "file" field values from the raw string
	// Pattern: "file":"<path>" or "file": "<path>"
	results := make(map[string]bool)

	idx := 0
	for {
		keyIdx := indexAfter(spec, `"file"`, idx)
		if keyIdx < 0 {
			break
		}
		// Find the colon after file key
		colonIdx := indexAfter(spec, ":", keyIdx+6)
		if colonIdx < 0 {
			break
		}
		// Find the opening quote
		quoteIdx := indexAfterChar(spec, '"', colonIdx+1)
		if quoteIdx < 0 {
			break
		}
		// Find the closing quote
		endIdx := -1
		for i := quoteIdx + 1; i < len(spec); i++ {
			if spec[i] == '"' && (i == 0 || spec[i-1] != '\\') {
				endIdx = i
				break
			}
		}
		if endIdx < 0 {
			break
		}
		path := spec[quoteIdx+1 : endIdx]
		resolvedPath, err := validatePathWithAllowPaths(path, t.workspace, t.restrict, t.allowPaths)
		if err != nil {
			return "", fmt.Errorf("path %q: %w", path, err)
		}
		results[path] = true
		// Replace the path with resolved path in the original spec
		spec = spec[:quoteIdx+1] + resolvedPath + spec[endIdx:]
		idx = quoteIdx + 1 + len(resolvedPath)
		_ = results
	}

	return spec, nil
}

func indexAfter(s, substr string, start int) int {
	idx := indexAt(s, substr, start)
	if idx < 0 {
		return -1
	}
	return idx + len(substr)
}

func indexAt(s, substr string, start int) int {
	if start >= len(s) {
		return -1
	}
	for i := start; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

func indexAfterChar(s string, b byte, start int) int {
	for i := start; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

// toInt converts a JSON-decoded number (float64) to int
func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	case int64:
		return int(n), true
	case json.Number:
		i, err := n.Int64()
		if err != nil {
			return 0, false
		}
		return int(i), true
	default:
		return 0, false
	}
}
