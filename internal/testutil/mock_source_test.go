package testutil_test

import (
	"context"
	"errors"
	"testing"
	"time"

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

	got, err := src.Sync(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 item, got %d", len(got))
	}
	if src.SyncCalled != 1 {
		t.Errorf("expected SyncCalled=1, got %d", src.SyncCalled)
	}
	if src.LastSince != nil {
		t.Error("expected LastSince=nil when nil was passed")
	}
}

func TestMockSource_Sync_RecordsLastSince(t *testing.T) {
	src := &testutil.MockSource{NameVal: "test"}
	now := time.Now()
	src.Sync(context.Background(), &now)
	if src.LastSince == nil || !src.LastSince.Equal(now) {
		t.Errorf("expected LastSince=%v, got %v", now, src.LastSince)
	}
}

func TestMockSource_Sync_PropagatesError(t *testing.T) {
	syncErr := errors.New("rate limited")
	src := &testutil.MockSource{
		NameVal: "test",
		SyncErr: syncErr,
	}

	_, err := src.Sync(context.Background(), nil)
	if !errors.Is(err, syncErr) {
		t.Errorf("expected %v, got %v", syncErr, err)
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
