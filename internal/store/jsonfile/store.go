// Package jsonfile provides a JSON file-backed implementation of core.Store.
// All state is kept in a single JSON file and flushed on every mutation.
// Suitable for development and testing; replace with SQLite for production use.
package jsonfile

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/theakshaypant/mission-control/internal/core"
)

// db is the in-memory representation of the JSON file.
type db struct {
	Items       map[string]core.Item      `json:"items"`
	SyncCursors map[string]time.Time      `json:"sync_cursors"`
	ItemStates  map[string]core.ItemState `json:"item_states"`
}

// Store implements core.Store using a single JSON file.
type Store struct {
	mu   sync.RWMutex
	path string
	data db
}

var _ core.Store = (*Store)(nil)

// Open loads (or creates) a JSON store at the given path.
func Open(path string) (core.Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("jsonfile store: create directory: %w", err)
	}

	s := &Store{
		path: path,
		data: db{
			Items:       make(map[string]core.Item),
			SyncCursors: make(map[string]time.Time),
			ItemStates:  make(map[string]core.ItemState),
		},
	}

	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("jsonfile store: read file: %w", err)
	}
	if err := json.Unmarshal(raw, &s.data); err != nil {
		return nil, fmt.Errorf("jsonfile store: parse file: %w", err)
	}
	if s.data.Items == nil {
		s.data.Items = make(map[string]core.Item)
	}
	if s.data.SyncCursors == nil {
		s.data.SyncCursors = make(map[string]time.Time)
	}
	if s.data.ItemStates == nil {
		s.data.ItemStates = make(map[string]core.ItemState)
	}
	return s, nil
}

// flush writes the in-memory state to disk. Must be called with the write lock held.
func (s *Store) flush() error {
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("jsonfile store: marshal: %w", err)
	}
	if err := os.WriteFile(s.path, raw, 0o600); err != nil {
		return fmt.Errorf("jsonfile store: write file: %w", err)
	}
	return nil
}

// UpsertItem inserts or updates an item by its ID.
func (s *Store) UpsertItem(_ context.Context, item core.Item) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Items[item.ID] = item
	return s.flush()
}

// ListItems returns items matching the filter. When NeedsAttention is true,
// only items that pass NeedsAttention (given their stored state) are returned.
func (s *Store) ListItems(_ context.Context, filter core.ItemFilter) ([]core.Item, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var out []core.Item
	for _, item := range s.data.Items {
		if filter.Source != "" && item.Source != filter.Source {
			continue
		}
		if len(filter.Types) > 0 && !containsType(filter.Types, item.Type) {
			continue
		}
		if filter.WaitsOnMe && !item.WaitsOnMe {
			continue
		}
		if filter.NeedsAttention {
			state := s.stateFor(item.ID)
			if !item.NeedsAttention(state) {
				continue
			}
		}
		out = append(out, item)
	}
	return out, nil
}

// GetLastSyncedAt returns the last successful sync time for a source.
func (s *Store) GetLastSyncedAt(_ context.Context, sourceName string) (*time.Time, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.data.SyncCursors[sourceName]
	if !ok {
		return nil, nil
	}
	return &t, nil
}

// SetLastSyncedAt records the sync time for a source.
func (s *Store) SetLastSyncedAt(_ context.Context, sourceName string, t time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.SyncCursors[sourceName] = t
	return s.flush()
}

// GetItemState returns the state for an item, or nil if none has been recorded.
func (s *Store) GetItemState(_ context.Context, itemID string) (*core.ItemState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stateFor(itemID), nil
}

// SetItemState writes the state for an item, replacing any existing state.
func (s *Store) SetItemState(_ context.Context, state core.ItemState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.ItemStates[state.ItemID] = state
	return s.flush()
}

// stateFor returns a pointer to the stored state for itemID, or nil if absent.
// Must be called with at least a read lock held.
func (s *Store) stateFor(itemID string) *core.ItemState {
	st, ok := s.data.ItemStates[itemID]
	if !ok {
		return nil
	}
	return &st
}

// containsType reports whether t is in types.
func containsType(types []core.ItemType, t core.ItemType) bool {
	for _, v := range types {
		if v == t {
			return true
		}
	}
	return false
}
