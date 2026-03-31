package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/theakshaypant/mission-control/internal/core"
)

// ItemSummary is a projection of core.Item for list and summary views.
// It contains the fields relevant for display and triage; source-specific
// attributes are omitted and can be fetched separately if needed.
type ItemSummary struct {
	ID            string          `json:"id"`
	Source        core.SourceKind `json:"source"`
	SourceName    string          `json:"source_name"`
	Type          core.ItemType   `json:"type"`
	Title         string          `json:"title"`
	URL           string          `json:"url"`
	Namespace     string          `json:"namespace"`
	UpdatedAt     time.Time       `json:"updated_at"`
	IsAssigned    bool            `json:"is_assigned"`
	ActiveSignals []string        `json:"active_signals,omitempty"`
}

// ListItems returns items matching filter, projected as ItemSummary values.
func (a *Actions) ListItems(ctx context.Context, filter core.ItemFilter) ([]ItemSummary, error) {
	items, err := a.store.ListItems(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list items: %w", err)
	}
	summaries := make([]ItemSummary, len(items))
	for i, item := range items {
		summaries[i] = toSummary(item)
	}
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].UpdatedAt.After(summaries[j].UpdatedAt)
	})
	return summaries, nil
}

// Summary returns items that currently need the user's attention.
func (a *Actions) Summary(ctx context.Context) ([]ItemSummary, error) {
	return a.ListItems(ctx, core.ItemFilter{NeedsAttention: true})
}

// DismissItem permanently hides an item from attention lists.
// Returns ErrNotFound (wrapped) if the item does not exist in the store.
func (a *Actions) DismissItem(ctx context.Context, id string) error {
	if err := a.requireItem(ctx, id); err != nil {
		return err
	}
	state, err := a.store.GetItemState(ctx, id)
	if err != nil {
		return fmt.Errorf("dismiss %s: get state: %w", id, err)
	}
	if state == nil {
		state = &core.ItemState{ItemID: id}
	}
	state.Dismissed = true
	if err := a.store.SetItemState(ctx, *state); err != nil {
		return fmt.Errorf("dismiss %s: set state: %w", id, err)
	}
	return nil
}

// SnoozeItem suppresses attention signals for an item until the given time.
// Returns ErrNotFound (wrapped) if the item does not exist in the store.
func (a *Actions) SnoozeItem(ctx context.Context, id string, until time.Time) error {
	if err := a.requireItem(ctx, id); err != nil {
		return err
	}
	state, err := a.store.GetItemState(ctx, id)
	if err != nil {
		return fmt.Errorf("snooze %s: get state: %w", id, err)
	}
	if state == nil {
		state = &core.ItemState{ItemID: id}
	}
	state.SnoozedUntil = &until
	if err := a.store.SetItemState(ctx, *state); err != nil {
		return fmt.Errorf("snooze %s: set state: %w", id, err)
	}
	return nil
}

// requireItem returns ErrNotFound (wrapped) if no item with id exists.
func (a *Actions) requireItem(ctx context.Context, id string) error {
	items, err := a.store.ListItems(ctx, core.ItemFilter{})
	if err != nil {
		return fmt.Errorf("check item %s: %w", id, err)
	}
	for _, item := range items {
		if item.ID == id {
			return nil
		}
	}
	return fmt.Errorf("%w: %s", ErrNotFound, id)
}

func toSummary(item core.Item) ItemSummary {
	s := ItemSummary{
		ID:         item.ID,
		Source:     item.Source,
		SourceName: item.SourceName,
		Type:       item.Type,
		Title:      item.Title,
		URL:        item.URL,
		Namespace:  item.Namespace,
		UpdatedAt:  item.UpdatedAt,
		IsAssigned: item.IsAssigned,
	}
	if len(item.Attributes) > 0 {
		var attrs struct {
			ActiveSignals []string `json:"active_signals"`
		}
		if err := json.Unmarshal(item.Attributes, &attrs); err == nil {
			s.ActiveSignals = attrs.ActiveSignals
		}
	}
	return s
}
