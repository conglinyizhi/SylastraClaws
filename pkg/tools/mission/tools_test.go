package mission

import (
	"context"
	"testing"
)

func newTestStoreNoLoad(t *testing.T) *Store {
	t.Helper()
	return newTestStore(t)
}

func TestAddTool_Name(t *testing.T) {
	if got := NewAddTool(nil).Name(); got != "task_add" {
		t.Errorf("expected 'task_add', got %q", got)
	}
}

func TestAddTool_Execute(t *testing.T) {
	t.Run("add with valid params", func(t *testing.T) {
		s := newTestStoreNoLoad(t)
		tool := NewAddTool(s)
		result := tool.Execute(context.Background(), map[string]any{
			"title":       "test task",
			"description": "a description",
			"priority":    float64(1),
			"status":      "pending",
		})
		if result.IsError {
			t.Fatalf("unexpected error: %s", result.ForLLM)
		}
		if result.ForLLM == "" {
			t.Fatal("expected non-empty ForLLM")
		}
	})

	t.Run("add with required only", func(t *testing.T) {
		s := newTestStoreNoLoad(t)
		tool := NewAddTool(s)
		result := tool.Execute(context.Background(), map[string]any{
			"title": "minimal",
		})
		if result.IsError {
			t.Fatalf("unexpected error: %s", result.ForLLM)
		}
	})

	t.Run("add with empty title errors", func(t *testing.T) {
		s := newTestStoreNoLoad(t)
		tool := NewAddTool(s)
		result := tool.Execute(context.Background(), map[string]any{
			"title": "   ",
		})
		if !result.IsError {
			t.Fatal("expected error for whitespace-only title")
		}
	})

	t.Run("add missing title errors", func(t *testing.T) {
		s := newTestStoreNoLoad(t)
		tool := NewAddTool(s)
		result := tool.Execute(context.Background(), map[string]any{})
		if !result.IsError {
			t.Fatal("expected error for missing title")
		}
	})

	t.Run("add with default priority", func(t *testing.T) {
		s := newTestStoreNoLoad(t)
		tool := NewAddTool(s)
		result := tool.Execute(context.Background(), map[string]any{
			"title": "default pri",
		})
		if result.IsError {
			t.Fatalf("unexpected error: %s", result.ForLLM)
		}
		// ForLLM should contain "pri:2"
		if !contains(result.ForLLM, "pri:2") {
			t.Errorf("expected default priority 2 in ForLLM, got %q", result.ForLLM)
		}
	})
}

func TestUpdateTool_Name(t *testing.T) {
	if got := NewUpdateTool(nil).Name(); got != "task_up" {
		t.Errorf("expected 'task_up', got %q", got)
	}
}

func TestUpdateTool_Execute(t *testing.T) {
	t.Run("update existing item", func(t *testing.T) {
		s := newTestStoreNoLoad(t)
		mustAdd(t, s, "original")
		tool := NewUpdateTool(s)
		result := tool.Execute(context.Background(), map[string]any{
			"id":     float64(1),
			"title":  "updated",
			"status": "done",
		})
		if result.IsError {
			t.Fatalf("unexpected error: %s", result.ForLLM)
		}
		if !contains(result.ForLLM, "updated") {
			t.Errorf("expected 'updated' in ForLLM, got %q", result.ForLLM)
		}
	})

	t.Run("update missing id errors", func(t *testing.T) {
		s := newTestStoreNoLoad(t)
		tool := NewUpdateTool(s)
		result := tool.Execute(context.Background(), map[string]any{
			"title": "no id",
		})
		if !result.IsError {
			t.Fatal("expected error for missing id")
		}
	})

	t.Run("update non-existent id returns error", func(t *testing.T) {
		s := newTestStoreNoLoad(t)
		tool := NewUpdateTool(s)
		result := tool.Execute(context.Background(), map[string]any{
			"id":    float64(999),
			"title": "ghost",
		})
		if !result.IsError {
			t.Fatal("expected error for non-existent id")
		}
	})
}

func TestRemoveTool_Name(t *testing.T) {
	if got := NewRemoveTool(nil).Name(); got != "task_rm" {
		t.Errorf("expected 'task_rm', got %q", got)
	}
}

func TestRemoveTool_Execute(t *testing.T) {
	t.Run("remove existing item", func(t *testing.T) {
		s := newTestStoreNoLoad(t)
		mustAdd(t, s, "to remove")
		tool := NewRemoveTool(s)
		result := tool.Execute(context.Background(), map[string]any{
			"id": float64(1),
		})
		if result.IsError {
			t.Fatalf("unexpected error: %s", result.ForLLM)
		}
		if !contains(result.ForLLM, "to remove") {
			t.Errorf("expected 'to remove' in ForLLM, got %q", result.ForLLM)
		}
		if len(s.List()) != 0 {
			t.Errorf("expected store to have 0 items after remove, got %d", len(s.List()))
		}
	})

	t.Run("remove missing id errors", func(t *testing.T) {
		s := newTestStoreNoLoad(t)
		tool := NewRemoveTool(s)
		result := tool.Execute(context.Background(), map[string]any{})
		if !result.IsError {
			t.Fatal("expected error for missing id")
		}
	})

	t.Run("remove non-existent id returns error", func(t *testing.T) {
		s := newTestStoreNoLoad(t)
		tool := NewRemoveTool(s)
		result := tool.Execute(context.Background(), map[string]any{
			"id": float64(999),
		})
		if !result.IsError {
			t.Fatal("expected error for non-existent id")
		}
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
