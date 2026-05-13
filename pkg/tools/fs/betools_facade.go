package fstools

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"

	betools "github.com/conglinyizhi/better-edit-tools-mcp/pkg/betools"
)

// BetterShowTool implements the Tool interface for betools Show operation.
type BetterShowTool struct {
	workspace  string
	restrict   bool
	allowPaths []*regexp.Regexp
}

func NewBetterShowTool(workspace string, restrict bool, allowPaths ...[]*regexp.Regexp) *BetterShowTool {
	var patterns []*regexp.Regexp
	if len(allowPaths) > 0 {
		patterns = allowPaths[0]
	}
	return &BetterShowTool{
		workspace:  workspace,
		restrict:   restrict,
		allowPaths: patterns,
	}
}

func (t *BetterShowTool) Name() string {
	return "better_show"
}

func (t *BetterShowTool) Description() string {
	return "Display file content with line numbers using better-edit-tools. Supports range display (start/end) and auto-mode for detecting function scope."
}

func (t *BetterShowTool) Parameters() map[string]any {
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
				"default":     1,
			},
			"end": map[string]any{
				"anyOf": []any{
					map[string]any{"type": "integer", "description": "Ending line number (inclusive)."},
					map[string]any{"type": "string", "enum": []string{"auto"}, "description": "Auto-detect range around cursor line."},
				},
				"description": "Ending line number or 'auto' for function-scope detection.",
			},
		},
		"required": []string{"file"},
	}
}

func (t *BetterShowTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	path, ok := args["file"].(string)
	if !ok || path == "" {
		// also check "path"
		path, ok = args["path"].(string)
		if !ok || path == "" {
			return ErrorResult("file is required")
		}
	}

	resolved, err := validatePathWithAllowPaths(path, t.workspace, t.restrict, t.allowPaths)
	if err != nil {
		return ErrorResult(fmt.Sprintf("invalid path: %v", err))
	}

	start := 1
	if v, ok := args["start"]; ok {
		start = toInt(v, 1)
	}

	endLine := 0 // default: auto
	if v, ok := args["end"]; ok {
		switch val := v.(type) {
		case float64:
			endLine = int(val)
		case string:
			if val != "auto" {
				endLine, _ = strconv.Atoi(val)
			}
		}
	}

	result, sessionID, err := betools.Show(resolved, start, endLine)
	if err != nil {
		return ErrorResult(err.Error())
	}

	output := fmt.Sprintf("File: %s (lines %d-%d / %d)\n\n%s", result.File, result.Start, result.End, result.Total, result.Content)
	if sessionID != "" {
		output = fmt.Sprintf("Session: %s\n\n%s", sessionID, output)
	}
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
	return "Replace a precise line range in a file with new content. Supports preview mode and format selection."
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
				"description": "Starting line number (1-indexed) of the range to replace.",
			},
			"end": map[string]any{
				"type":        "integer",
				"description": "Ending line number (inclusive) of the range to replace.",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "The new content to insert. Supports JSON escaping (\\n for newline).",
			},
			"old": map[string]any{
				"type":        "string",
				"description": "Optional: exact old content to validate against. If provided, the tool verifies the current content matches before replacing.",
			},
			"format": map[string]any{
				"type":        "string",
				"description": "Diff format: 'full' (default) or 'compact'.",
				"default":     "full",
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
	path, ok := args["file"].(string)
	if !ok || path == "" {
		return ErrorResult("file is required")
	}

	resolved, err := validatePathWithAllowPaths(path, t.workspace, t.restrict, t.allowPaths)
	if err != nil {
		return ErrorResult(fmt.Sprintf("invalid path: %v", err))
	}

	start := toInt(args["start"], 0)
	if start < 1 {
		return ErrorResult("start is required and must be >= 1")
	}

	end := toInt(args["end"], 0)
	if end < start {
		return ErrorResult("end must be >= start")
	}

	content, _ := args["content"].(string)
	raw := false
	if v, _ := args["raw"].(bool); v {
		raw = true
	}
	format, _ := args["format"].(string)
	preview, _ := args["preview"].(bool)

	var old *string
	if v, ok := args["old"].(string); ok && v != "" {
		old = &v
	}

	result, err := betools.Replace(resolved, start, end, old, content, raw, format, preview, "")
	if err != nil {
		return ErrorResult(err.Error())
	}

	output := fmt.Sprintf("File: %s\nStatus: %s\nAffected: %s\nDiff:\n%s",
		result.File, result.Status, result.Affected, result.Diff)
	if result.Warning != "" {
		output += "\nWarning: " + result.Warning
	}
	if preview {
		return NewToolResult("[PREVIEW - no changes applied]\n\n" + output)
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
	return "Insert content after a specific line in a file. Line 0 means insert at the beginning."
}

func (t *BetterInsertTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"file": map[string]any{
				"type":        "string",
				"description": "The file path to insert into.",
			},
			"line": map[string]any{
				"type":        "integer",
				"description": "Insert content after this line number. 0 = insert at beginning.",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "The content to insert. Supports JSON escaping (\\n for newline).",
			},
			"format": map[string]any{
				"type":        "string",
				"description": "Diff format: 'full' (default) or 'compact'.",
				"default":     "full",
			},
			"preview": map[string]any{
				"type":        "boolean",
				"description": "If true, only show the diff without applying changes.",
				"default":     false,
			},
		},
		"required": []string{"file", "line", "content"},
	}
}

func (t *BetterInsertTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	path, ok := args["file"].(string)
	if !ok || path == "" {
		return ErrorResult("file is required")
	}

	resolved, err := validatePathWithAllowPaths(path, t.workspace, t.restrict, t.allowPaths)
	if err != nil {
		return ErrorResult(fmt.Sprintf("invalid path: %v", err))
	}

	after := toInt(args["line"], 0)
	content, _ := args["content"].(string)
	raw, _ := args["raw"].(bool)
	format, _ := args["format"].(string)
	preview, _ := args["preview"].(bool)

	result, err := betools.Insert(resolved, after, content, raw, format, preview)
	if err != nil {
		return ErrorResult(err.Error())
	}

	output := fmt.Sprintf("File: %s\nStatus: %s\nAfter line: %d\nAffected: %s\nDiff:\n%s",
		result.File, result.Status, result.After, result.Affected, result.Diff)
	if preview {
		return NewToolResult("[PREVIEW - no changes applied]\n\n" + output)
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
	return "Delete one line, a line range, or a batch of specific line numbers from a file."
}

func (t *BetterDeleteTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"file": map[string]any{
				"type":        "string",
				"description": "The file path to delete lines from.",
			},
			"start": map[string]any{
				"type":        "integer",
				"description": "Starting line number of the range to delete (1-indexed).",
			},
			"end": map[string]any{
				"type":        "integer",
				"description": "Ending line number (inclusive) of the range to delete.",
			},
			"line": map[string]any{
				"type":        "integer",
				"description": "Single line number to delete (alternative to start/end range).",
			},
			"lines": map[string]any{
				"type":        "string",
				"description": "JSON array of specific line numbers to delete, e.g. '[3, 7, 12]'.",
			},
			"format": map[string]any{
				"type":        "string",
				"description": "Diff format: 'full' (default) or 'compact'.",
				"default":     "full",
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
	path, ok := args["file"].(string)
	if !ok || path == "" {
		return ErrorResult("file is required")
	}

	resolved, err := validatePathWithAllowPaths(path, t.workspace, t.restrict, t.allowPaths)
	if err != nil {
		return ErrorResult(fmt.Sprintf("invalid path: %v", err))
	}

	start := toInt(args["start"], 0)
	end := toInt(args["end"], 0)
	line := toInt(args["line"], 0)
	format, _ := args["format"].(string)
	preview, _ := args["preview"].(bool)

	var linesJSON *string
	if v, ok := args["lines"].(string); ok && v != "" {
		linesJSON = &v
	}

	result, err := betools.Delete(resolved, start, end, line, linesJSON, format, preview)
	if err != nil {
		return ErrorResult(err.Error())
	}

	output := fmt.Sprintf("File: %s\nStatus: %s\nAffected: %s\nDiff:\n%s",
		result.File, result.Status, result.Affected, result.Diff)
	if preview {
		return NewToolResult("[PREVIEW - no changes applied]\n\n" + output)
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
	return "Apply multiple edits in one call, including multi-file edits. Edits are sorted bottom-to-top automatically to preserve line numbers."
}

func (t *BetterBatchTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"spec": map[string]any{
				"type":        "string",
				"description": "JSON specification. Can be a single file object or array of file objects. Each file object has: 'file' (string), 'edits' (array of {action, start, end, line, content}). Actions: replace-lines, insert-after, delete-lines.",
			},
			"preview": map[string]any{
				"type":        "boolean",
				"description": "If true, only show what would be changed without applying.",
				"default":     false,
			},
		},
		"required": []string{"spec"},
	}
}

func (t *BetterBatchTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	specRaw, ok := args["spec"].(string)
	if !ok || specRaw == "" {
		return ErrorResult("spec is required")
	}

	preview, _ := args["preview"].(bool)

	// Resolve file paths in the spec to workspace-relative paths
	// We need to parse the spec, resolve paths, then pass to betools
	var rawVal any
	if err := json.Unmarshal([]byte(specRaw), &rawVal); err != nil {
		return ErrorResult(fmt.Sprintf("invalid JSON spec: %v", err))
	}

	// Walk the spec and resolve file paths through workspace validation
	resolvedSpec, err := t.resolveSpecPaths(rawVal)
	if err != nil {
		return ErrorResult(err.Error())
	}

	resolvedJSON, err := json.Marshal(resolvedSpec)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to marshal resolved spec: %v", err))
	}

	result, err := betools.Batch(string(resolvedJSON), preview)
	if err != nil {
		return ErrorResult(err.Error())
	}

	output := fmt.Sprintf("Status: %s\nFiles affected: %d\n", result.Status, result.Files)
	for _, r := range result.Results {
		output += fmt.Sprintf("  - %s: %d edits, %d total lines\n", r.File, r.Edits, r.Total)
	}
	if preview {
		return NewToolResult("[PREVIEW - no changes applied]\n\n" + output)
	}
	return NewToolResult(output)
}

func (t *BetterBatchTool) resolveSpecPaths(raw any) (any, error) {
	switch v := raw.(type) {
	case []any:
		out := make([]any, 0, len(v))
		for _, item := range v {
			resolved, err := t.resolveSpecPaths(item)
			if err != nil {
				return nil, err
			}
			out = append(out, resolved)
		}
		return out, nil
	case map[string]any:
		out := make(map[string]any, len(v))
		for k, val := range v {
			if k == "file" {
				path, ok := val.(string)
				if ok && path != "" {
					resolved, err := validatePathWithAllowPaths(path, t.workspace, t.restrict, t.allowPaths)
					if err != nil {
						return nil, fmt.Errorf("invalid path %q: %v", path, err)
					}
					out[k] = resolved
					continue
				}
			}
			resolved, err := t.resolveSpecPaths(val)
			if err != nil {
				return nil, err
			}
			out[k] = resolved
		}
		return out, nil
	default:
		return raw, nil
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
	return "Write raw content to one or more files. Supports multi-file writes with atomic operations and degraded parsing for malformed JSON payloads."
}

func (t *BetterWriteTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"file": map[string]any{
				"type":        "string",
				"description": "The file path to write to (single file mode).",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "The content to write (single file mode).",
			},
			"files": map[string]any{
				"type":        "array",
				"description": "Array of {file, content} objects for multi-file writes.",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"file": map[string]any{
							"type":        "string",
							"description": "File path.",
						},
						"content": map[string]any{
							"type":        "string",
							"description": "File content.",
						},
					},
					"required": []string{"file", "content"},
				},
			},
			"preview": map[string]any{
				"type":        "boolean",
				"description": "If true, show what would be written without writing.",
				"default":     false,
			},
			"raw": map[string]any{
				"type":        "boolean",
				"description": "If true, treat \\n as literal newlines without JSON escaping.",
				"default":     false,
			},
		},
		"oneOf": []any{
			map[string]any{
				"title": 	   "single",
				"required":    []string{"file", "content"},
			},
			map[string]any{
				"title":       "multi",
				"required":    []string{"files"},
			},
		},
	}
}

func (t *BetterWriteTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	preview, _ := args["preview"].(bool)
	raw, _ := args["raw"].(bool)

	// Build write spec
	spec := buildWriteSpec(args)
	if spec == "" {
		return ErrorResult("write requires either 'file'/'content' or 'files' array")
	}

	// Resolve file paths in the spec
	var rawVal any
	if err := json.Unmarshal([]byte(spec), &rawVal); err != nil {
		return ErrorResult(fmt.Sprintf("invalid spec: %v", err))
	}
	// Use resolveSpecPaths from BetterBatchTool logic
	resolvedSpec, err := t.resolveWritePaths(rawVal)
	if err != nil {
		return ErrorResult(err.Error())
	}
	resolvedJSON, err := json.Marshal(resolvedSpec)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to marshal resolved spec: %v", err))
	}

	result, err := betools.Write(string(resolvedJSON), preview, raw)
	if err != nil {
		return ErrorResult(err.Error())
	}

	output := fmt.Sprintf("Status: %s\nFiles written: %d\n", result.Status, result.Files)
	for _, r := range result.Results {
		output += fmt.Sprintf("  - %s: %d lines, %d bytes\n", r.File, r.Lines, r.Bytes)
	}
	if result.Degraded {
		output += "\n⚠ Degraded parsing: " + result.Warning
	}
	if preview {
		return NewToolResult("[PREVIEW - no changes applied]\n\n" + output)
	}
	return NewToolResult(output)
}

func (t *BetterWriteTool) resolveWritePaths(raw any) (any, error) {
	switch v := raw.(type) {
	case []any:
		out := make([]any, 0, len(v))
		for _, item := range v {
			resolved, err := t.resolveWritePaths(item)
			if err != nil {
				return nil, err
			}
			out = append(out, resolved)
		}
		return out, nil
	case map[string]any:
		out := make(map[string]any, len(v))
		for k, val := range v {
			if k == "file" {
				path, ok := val.(string)
				if ok && path != "" {
					resolved, err := validatePathWithAllowPaths(path, t.workspace, t.restrict, t.allowPaths)
					if err != nil {
						return nil, fmt.Errorf("invalid path %q: %v", path, err)
					}
					out[k] = resolved
					continue
				}
			}
			resolved, err := t.resolveWritePaths(val)
			if err != nil {
				return nil, err
			}
			out[k] = resolved
		}
		return out, nil
	default:
		return raw, nil
	}
}

func buildWriteSpec(args map[string]any) string {
	// Check for files array first
	if files, ok := args["files"].([]any); ok && len(files) > 0 {
		spec := map[string]any{"files": files}
		data, _ := json.Marshal(spec)
		return string(data)
	}

	// Single file mode
	file, _ := args["file"].(string)
	content, _ := args["content"].(string)
	if file != "" {
		spec := map[string]any{"file": file, "content": content}

		// Also add extract if present
		if extract, ok := args["extract"]; ok {
			spec["extract"] = extract
		}

		data, _ := json.Marshal(spec)
		return string(data)
	}

	return ""
}

// Helper: convert any value to int
func toInt(v any, defaultVal int) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	case float32:
		return int(n)
	case json.Number:
		if i, err := n.Int64(); err == nil {
			return int(i)
		}
	}
	return defaultVal
}
