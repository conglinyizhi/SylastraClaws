package mission

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
)

// newTestStore creates a Store backed by MemMapFs.
func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := NewStoreWithFS(afero.NewMemMapFs(), "/test/missions.json")
	if err != nil {
		t.Fatalf("NewStoreWithFS: %v", err)
	}
	return s
}

func mustAdd(t *testing.T, s *Store, title string) MissionItem {
	t.Helper()
	item, err := s.Add(title, "", "", 2)
	if err != nil {
		t.Fatalf("Add(%q): %v", title, err)
	}
	return item
}

func TestStore_Add(t *testing.T) {
	t.Run("basic add", func(t *testing.T) {
		s := newTestStore(t)
		item, err := s.Add("test task", "some description", "pending", 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if item.ID != 1 {
			t.Errorf("expected ID 1, got %d", item.ID)
		}
		if item.Title != "test task" {
			t.Errorf("expected title 'test task', got %q", item.Title)
		}
		if item.Priority != 1 {
			t.Errorf("expected priority 1, got %d", item.Priority)
		}
		if item.Status != "pending" {
			t.Errorf("expected status 'pending', got %q", item.Status)
		}
		if item.Description != "some description" {
			t.Errorf("expected description 'some description', got %q", item.Description)
		}
		if item.CreatedAt.IsZero() {
			t.Error("expected non-zero CreatedAt")
		}
	})

	t.Run("auto ID increment", func(t *testing.T) {
		s := newTestStore(t)
		a1 := mustAdd(t, s, "first")
		a2 := mustAdd(t, s, "second")
		a3 := mustAdd(t, s, "third")
		if a1.ID != 1 || a2.ID != 2 || a3.ID != 3 {
			t.Errorf("expected IDs 1,2,3; got %d,%d,%d", a1.ID, a2.ID, a3.ID)
		}
	})

	t.Run("default status", func(t *testing.T) {
		s := newTestStore(t)
		item, err := s.Add("no status", "", "", 2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if item.Status != "pending" {
			t.Errorf("expected default status 'pending', got %q", item.Status)
		}
	})

	t.Run("empty title allowed", func(t *testing.T) {
		s := newTestStore(t)
		item, err := s.Add("", "some description", "done", 2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if item.Title != "" {
			t.Errorf("expected empty title, got %q", item.Title)
		}
	})
}

func TestStore_Update(t *testing.T) {
	t.Run("update all fields", func(t *testing.T) {
		s := newTestStore(t)
		mustAdd(t, s, "original")
		prio := 1
		updated, found, err := s.Update(1, "updated title", "updated desc", "done", &prio)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !found {
			t.Fatal("expected item to be found")
		}
		if updated.Title != "updated title" {
			t.Errorf("expected title 'updated title', got %q", updated.Title)
		}
		if updated.Description != "updated desc" {
			t.Errorf("expected desc 'updated desc', got %q", updated.Description)
		}
		if updated.Status != "done" {
			t.Errorf("expected status 'done', got %q", updated.Status)
		}
		if updated.Priority != 1 {
			t.Errorf("expected priority 1, got %d", updated.Priority)
		}
	})

	t.Run("partial update only title", func(t *testing.T) {
		s := newTestStore(t)
		mustAdd(t, s, "original")
		updated, found, err := s.Update(1, "new title", "", "", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !found {
			t.Fatal("expected item to be found")
		}
		if updated.Title != "new title" {
			t.Errorf("expected title 'new title', got %q", updated.Title)
		}
		if updated.Status != "pending" {
			t.Errorf("expected status unchanged 'pending', got %q", updated.Status)
		}
		if updated.Priority != 2 {
			t.Errorf("expected priority unchanged 2, got %d", updated.Priority)
		}
	})

	t.Run("update non-existent ID", func(t *testing.T) {
		s := newTestStore(t)
		_, found, err := s.Update(99, "nope", "", "", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found {
			t.Fatal("expected not found for non-existent ID")
		}
	})

	t.Run("update preserves CreatedAt", func(t *testing.T) {
		s := newTestStore(t)
		original := mustAdd(t, s, "original")
		prio := 3
		updated, _, err := s.Update(1, "", "", "done", &prio)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !updated.CreatedAt.Equal(original.CreatedAt) {
			t.Error("CreatedAt should not change on update")
		}
		if updated.UpdatedAt.Before(original.UpdatedAt) {
			t.Error("UpdatedAt should advance on update")
		}
	})
}

func TestStore_Remove(t *testing.T) {
	t.Run("remove existing item", func(t *testing.T) {
		s := newTestStore(t)
		mustAdd(t, s, "to remove")
		removed, found, err := s.Remove(1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !found {
			t.Fatal("expected item to be found")
		}
		if removed.Title != "to remove" {
			t.Errorf("expected title 'to remove', got %q", removed.Title)
		}
		if len(s.List()) != 0 {
			t.Errorf("expected 0 items after remove, got %d", len(s.List()))
		}
	})

	t.Run("remove non-existent ID", func(t *testing.T) {
		s := newTestStore(t)
		mustAdd(t, s, "keep me")
		_, found, err := s.Remove(99)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found {
			t.Fatal("expected not found for non-existent ID")
		}
		if len(s.List()) != 1 {
			t.Errorf("expected 1 item remaining, got %d", len(s.List()))
		}
	})

	t.Run("remove preserves remaining items", func(t *testing.T) {
		s := newTestStore(t)
		a1 := mustAdd(t, s, "first")
		a2 := mustAdd(t, s, "second")
		a3 := mustAdd(t, s, "third")

		s.Remove(a2.ID)

		items := s.List()
		if len(items) != 2 {
			t.Fatalf("expected 2 items, got %d", len(items))
		}
		if items[0].ID != a1.ID || items[1].ID != a3.ID {
			t.Errorf("expected IDs [%d,%d], got [%d,%d]", a1.ID, a3.ID, items[0].ID, items[1].ID)
		}
	})
}

func TestStore_List(t *testing.T) {
	t.Run("empty store", func(t *testing.T) {
		s := newTestStore(t)
		items := s.List()
		if len(items) != 0 {
			t.Errorf("expected 0 items, got %d", len(items))
		}
	})

	t.Run("returns sorted by ID", func(t *testing.T) {
		s := newTestStore(t)
		_ = mustAdd(t, s, "A")
		_ = mustAdd(t, s, "B")
		_ = mustAdd(t, s, "C")

		items := s.List()
		if len(items) != 3 {
			t.Fatalf("expected 3 items, got %d", len(items))
		}
		for i, item := range items {
			if item.ID != i+1 {
				t.Errorf("item %d expected ID %d, got %d", i, i+1, item.ID)
			}
		}
	})

	t.Run("returns a copy, not a reference", func(t *testing.T) {
		s := newTestStore(t)
		mustAdd(t, s, "original")
		items := s.List()
		items[0].Title = "hacked"
		after := s.List()
		if after[0].Title == "hacked" {
			t.Error("List() should return a copy, mutating it should not affect store")
		}
	})
}

func TestStore_Persistence(t *testing.T) {
	t.Run("items survive load after add", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		path := "/test/missions.json"

		s1, err := NewStoreWithFS(fs, path)
		if err != nil {
			t.Fatalf("first NewStoreWithFS: %v", err)
		}
		mustAdd(t, s1, "persistent")

		s2, err := NewStoreWithFS(fs, path)
		if err != nil {
			t.Fatalf("second NewStoreWithFS: %v", err)
		}
		items := s2.List()
		if len(items) != 1 {
			t.Fatalf("expected 1 item after reload, got %d", len(items))
		}
		if items[0].Title != "persistent" {
			t.Errorf("expected title 'persistent', got %q", items[0].Title)
		}
	})

	t.Run("load handles non-existent file gracefully", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		s, err := NewStoreWithFS(fs, "/test/missions.json")
		if err != nil {
			t.Fatalf("NewStoreWithFS: %v", err)
		}
		items := s.List()
		if items == nil {
			t.Error("List() should return empty slice, not nil after fresh load")
		}
	})

	t.Run("atomic write leaves no tmp file", func(t *testing.T) {
		s := newTestStore(t)
		mustAdd(t, s, "atomic test")
		tmpPath := "/test/missions.json.tmp"
		exists, _ := afero.Exists(s.fs, tmpPath)
		if exists {
			t.Error("tmp file should be cleaned up after save")
		}
	})
}

func TestStore_Concurrency(t *testing.T) {
	s := newTestStore(t)

	t.Run("parallel adds", func(t *testing.T) {
		n := 10
		errs := make(chan error, n)
		for i := 0; i < n; i++ {
			go func(i int) {
				_, err := s.Add("goroutine", "", "", 2)
				errs <- err
			}(i)
		}
		for i := 0; i < n; i++ {
			if err := <-errs; err != nil {
				t.Errorf("concurrent add failed: %v", err)
			}
		}
		if len(s.List()) != n {
			t.Errorf("expected %d items after concurrent adds, got %d", n, len(s.List()))
		}
	})
}

func TestStore_findIndex(t *testing.T) {
	s := newTestStore(t)
	mustAdd(t, s, "A")
	mustAdd(t, s, "B")
	mustAdd(t, s, "C")

	tests := []struct {
		id    int
		index int
	}{
		{1, 0},
		{2, 1},
		{3, 2},
		{99, -1},
		{-1, -1},
	}

	for _, tt := range tests {
		got := s.findIndex(tt.id)
		if got != tt.index {
			t.Errorf("findIndex(%d) = %d, want %d", tt.id, got, tt.index)
		}
	}
}

func TestNewStore_RealFS(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missions.json")
	fs := afero.NewOsFs()

	s, err := NewStoreWithFS(fs, path)
	if err != nil {
		t.Fatalf("NewStoreWithFS: %v", err)
	}
	mustAdd(t, s, "real fs test")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected data on real fs")
	}
}
