package testutil_test

import (
	"context"
	"errors"
	"testing"

	"github.com/theakshaypant/mission-control/internal/core"
	"github.com/theakshaypant/mission-control/internal/testutil"
)

// Compile-time checks: mocks satisfy the core interfaces.
var _ core.Source = (*testutil.MockSource)(nil)
var _ core.SourceConfig = (*testutil.MockSourceConfig)(nil)

const (
	testKind core.SourceKind = "test"
	testType core.ItemType   = "task"
)

func TestMockSource_Sync_ReturnsItems(t *testing.T) {
	items := []core.Item{
		{ID: "test:ns#1", Source: testKind, Type: testType},
	}
	src := &testutil.MockSource{
		NameVal: "test",
		KindVal: testKind,
		Items:   items,
	}

	got, err := src.Sync(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 item, got %d", len(got))
	}
	if src.SyncCalled != 1 {
		t.Errorf("expected SyncCalled=1, got %d", src.SyncCalled)
	}
	if src.LastSyncedAt() == nil {
		t.Error("expected LastSyncedAt to be set after sync")
	}
}

func TestMockSource_Sync_PropagatesError(t *testing.T) {
	syncErr := errors.New("rate limited")
	src := &testutil.MockSource{
		NameVal: "test",
		SyncErr: syncErr,
	}

	_, err := src.Sync(context.Background())
	if !errors.Is(err, syncErr) {
		t.Errorf("expected %v, got %v", syncErr, err)
	}
	if src.LastSyncedAt() != nil {
		t.Error("LastSyncedAt should not be set on error")
	}
}

func TestMockSource_NeverSynced(t *testing.T) {
	src := &testutil.MockSource{NameVal: "test"}
	if src.LastSyncedAt() != nil {
		t.Error("expected nil LastSyncedAt before first sync")
	}
}

func TestMockSourceConfig_Validate(t *testing.T) {
	cfg := &testutil.MockSourceConfig{}
	if err := cfg.Validate(); err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	validationErr := errors.New("invalid")
	cfg.ValidationErr = validationErr
	if !errors.Is(cfg.Validate(), validationErr) {
		t.Errorf("expected validation error %v", validationErr)
	}
}
