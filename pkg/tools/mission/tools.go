package mission

import (
	"context"
	"fmt"
	"strings"

	"github.com/conglinyizhi/SylastraClaws/pkg/tools/shared"
)

// m_add: add a new mission item.
type AddTool struct {
	store *Store
}

func NewAddTool(s *Store) *AddTool {
	return &AddTool{store: s}
}

func (t *AddTool) Name() string   { return "m_add" }
func (t *AddTool) Description() string { return "Add a new mission/task item." }

func (t *AddTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"title": map[string]any{
				"type":        "string",
				"description": "Task title (required, short)",
			},
			"description": map[string]any{
				"type":        "string",
				"description": "Task description or notes (optional)",
			},
			"priority": map[string]any{
				"type":        "integer",
				"description": "Priority: 1=high, 2=medium, 3=low (default 2)",
			},
			"status": map[string]any{
				"type":        "string",
				"description": "Initial status: pending (default)",
				"enum":        []string{"pending", "done"},
			},
		},
		"required": []string{"title"},
	}
}

func (t *AddTool) Execute(ctx context.Context, args map[string]any) *toolshared.ToolResult {
	title, _ := args["title"].(string)
	title = strings.TrimSpace(title)
	if title == "" {
		return toolshared.ErrorResult("title is required")
	}

	desc, _ := args["description"].(string)
	status, _ := args["status"].(string)
	priority := 2
	if p, ok := args["priority"].(float64); ok {
		priority = int(p)
	}

	item, err := t.store.Add(title, desc, status, priority)
	if err != nil {
		return toolshared.ErrorResult("failed to add mission: " + err.Error())
	}

	return &toolshared.ToolResult{
		ForLLM: fmt.Sprintf("[pending] %s (id:%d pri:%d)", item.Title, item.ID, item.Priority),
	}
}

// m_up: update an existing mission item.
type UpdateTool struct {
	store *Store
}

func NewUpdateTool(s *Store) *UpdateTool {
	return &UpdateTool{store: s}
}

func (t *UpdateTool) Name() string   { return "m_up" }
func (t *UpdateTool) Description() string { return "Update a mission item's fields (status, title, priority)." }

func (t *UpdateTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id": map[string]any{
				"type":        "integer",
				"description": "Mission item ID",
			},
			"title": map[string]any{
				"type":        "string",
				"description": "New title (optional)",
			},
			"description": map[string]any{
				"type":        "string",
				"description": "New description (optional)",
			},
			"status": map[string]any{
				"type":        "string",
				"description": "New status",
				"enum":        []string{"pending", "done", "closed"},
			},
			"priority": map[string]any{
				"type":        "integer",
				"description": "New priority: 1=high, 2=medium, 3=low",
			},
		},
		"required": []string{"id"},
	}
}

func fmtItem(item MissionItem) string {
	return fmt.Sprintf("[%s] %s (id:%d pri:%d)", item.Status, item.Title, item.ID, item.Priority)
}

func (t *UpdateTool) Execute(ctx context.Context, args map[string]any) *toolshared.ToolResult {
	id, ok := args["id"].(float64)
	if !ok {
		return toolshared.ErrorResult("id is required and must be a number")
	}
	pid := int(id)

	title, _ := args["title"].(string)
	desc, _ := args["description"].(string)
	status, _ := args["status"].(string)
	var priority *int
	if p, ok := args["priority"].(float64); ok {
		v := int(p)
		priority = &v
	}

	item, found, err := t.store.Update(pid, title, desc, status, priority)
	if err != nil {
		return toolshared.ErrorResult("failed to update mission: " + err.Error())
	}
	if !found {
		return toolshared.ErrorResult(fmt.Sprintf("mission #%d not found", pid))
	}

	var changed []string
	if title != "" {
		changed = append(changed, "title")
	}
	if desc != "" {
		changed = append(changed, "description")
	}
	if status != "" {
		changed = append(changed, "status")
	}
	if priority != nil {
		changed = append(changed, "priority")
	}

	forLLM := fmt.Sprintf("updated #%d (%s): %s", pid, strings.Join(changed, ","), fmtItem(item))
	return &toolshared.ToolResult{ForLLM: forLLM}
}

// m_rm: remove a mission item by ID. Returns the full item so the LLM knows
// what was removed.
type RemoveTool struct {
	store *Store
}

func NewRemoveTool(s *Store) *RemoveTool {
	return &RemoveTool{store: s}
}

func (t *RemoveTool) Name() string   { return "m_rm" }
func (t *RemoveTool) Description() string { return "Remove a mission item by ID." }

func (t *RemoveTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id": map[string]any{
				"type":        "integer",
				"description": "Mission item ID to remove",
			},
		},
		"required": []string{"id"},
	}
}

func (t *RemoveTool) Execute(ctx context.Context, args map[string]any) *toolshared.ToolResult {
	id, ok := args["id"].(float64)
	if !ok {
		return toolshared.ErrorResult("id is required and must be a number")
	}

	item, found, err := t.store.Remove(int(id))
	if err != nil {
		return toolshared.ErrorResult("failed to remove mission: " + err.Error())
	}
	if !found {
		return toolshared.ErrorResult(fmt.Sprintf("mission #%d not found", int(id)))
	}

	return &toolshared.ToolResult{
		ForLLM: fmt.Sprintf("[deleted] %s", fmtItem(item)),
	}
}
