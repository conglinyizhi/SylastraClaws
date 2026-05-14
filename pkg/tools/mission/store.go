package mission

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Store manages mission items as a JSON file in XDG data home.
type Store struct {
	mu       sync.RWMutex
	filePath string
	items    []MissionItem
	nextID   int
}

// NewStore opens or creates the mission store.
func NewStore() (*Store, error) {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("cannot determine home directory: %w", err)
		}
		dataHome = filepath.Join(home, ".local", "share")
	}

	dir := filepath.Join(dataHome, "sylastraclaws")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("cannot create data dir %s: %w", dir, err)
	}

	s := &Store{
		filePath: filepath.Join(dir, "missions.json"),
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

// List returns a copy of all items sorted by ID ascending.
func (s *Store) List() []MissionItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]MissionItem, len(s.items))
	copy(out, s.items)
	return out
}

// Add inserts a new item and returns its assigned ID.
func (s *Store) Add(title, desc, status string, priority int) (MissionItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if status == "" {
		status = "pending"
	}
	item := MissionItem{
		ID:          s.nextID,
		Title:       title,
		Description: desc,
		Status:      status,
		Priority:    priority,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	s.nextID++
	s.items = append(s.items, item)

	if err := s.save(); err != nil {
		return MissionItem{}, err
	}
	return item, nil
}

// Update modifies fields of an existing item. Returns the updated item and
// whether the item existed.
func (s *Store) Update(id int, title, desc, status string, priority *int) (MissionItem, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.findIndex(id)
	if idx < 0 {
		return MissionItem{}, false, nil
	}

	item := &s.items[idx]
	if title != "" {
		item.Title = title
	}
	if desc != "" {
		item.Description = desc
	}
	if status != "" {
		item.Status = status
	}
	if priority != nil {
		item.Priority = *priority
	}
	item.UpdatedAt = time.Now()

	if err := s.save(); err != nil {
		return MissionItem{}, false, err
	}
	return *item, true, nil
}

// Remove deletes an item by ID. Returns the deleted item and whether it existed.
func (s *Store) Remove(id int) (MissionItem, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.findIndex(id)
	if idx < 0 {
		return MissionItem{}, false, nil
	}

	item := s.items[idx]
	s.items = append(s.items[:idx], s.items[idx+1:]...)

	if err := s.save(); err != nil {
		return MissionItem{}, false, err
	}
	return item, true, nil
}

func (s *Store) findIndex(id int) int {
	for i, v := range s.items {
		if v.ID == id {
			return i
		}
	}
	return -1
}

func (s *Store) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			s.items = nil
			s.nextID = 1
			return nil
		}
		return fmt.Errorf("cannot read mission file %s: %w", s.filePath, err)
	}

	var state struct {
		NextID int           `json:"next_id"`
		Items  []MissionItem `json:"items"`
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("cannot parse mission file %s: %w", s.filePath, err)
	}
	s.items = state.Items
	s.nextID = state.NextID
	if s.items == nil {
		s.items = []MissionItem{}
	}
	if s.nextID <= 0 {
		s.nextID = 1
	}

	// Keep items sorted by ID for consistency
	sort.Slice(s.items, func(i, j int) bool {
		return s.items[i].ID < s.items[j].ID
	})

	return nil
}

func (s *Store) save() error {
	state := struct {
		NextID int           `json:"next_id"`
		Items  []MissionItem `json:"items"`
	}{
		NextID: s.nextID,
		Items:  s.items,
	}

	data, err := json.MarshalIndent(state, "", "\t")
	if err != nil {
		return fmt.Errorf("cannot marshal missions: %w", err)
	}

	// Atomic write: write to tmp, then rename
	tmpPath := s.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("cannot write mission file %s: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, s.filePath); err != nil {
		return fmt.Errorf("cannot commit mission file %s: %w", s.filePath, err)
	}
	return nil
}
