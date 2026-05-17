package fstools

import (
	"context"
	"fmt"
	"strings"

	betools "github.com/conglinyizhi/better-edit-tools-mcp/pkg/betools"
)

// BetterRollbackTool implements the Tool interface for betools RollbackSnapshots operation.
type BetterRollbackTool struct{}

func NewBetterRollbackTool() *BetterRollbackTool {
	return &BetterRollbackTool{}
}

func (t *BetterRollbackTool) Name() string {
	return "trx-rollback"
}

func (t *BetterRollbackTool) Description() string {
	return "Roll back the last N file editing operations. Each tracked write/replace/insert/delete pushes a snapshot; this tool restores files to their state before those N operations. Use steps=1 for a single undo, or higher for batch undo. Returns the number of snapshots rolled back."
}

func (t *BetterRollbackTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"steps": map[string]any{
				"type":        "integer",
				"description": "Number of editing operations to roll back (1 = undo last edit, 2 = undo last two edits, etc.). Default: 1.",
				"default":     1,
			},
		},
		"required": []string{},
	}
}

func (t *BetterRollbackTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	steps := 1
	if s, ok := toInt(args["steps"]); ok && s > 0 {
		steps = s
	}

	count, errs := betools.RollbackSnapshots(steps)
	if count == 0 && len(errs) == 0 {
		return NewToolResult("Nothing to roll back \u2014 no snapshots in queue.")
	}

	var output string
	if count > 0 {
		output = fmt.Sprintf("Rolled back %d operation(s).\n", count)
	}
	if len(errs) > 0 {
		for i, err := range errs {
			output += fmt.Sprintf("  Error %d: %v\n", i+1, err)
		}
	}
	return NewToolResult(strings.TrimSpace(output))
}
