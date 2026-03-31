// Package testutil provides test helpers including a mock Source implementation.
package testutil

import (
	"context"
	"time"

	"github.com/theakshaypant/mission-control/internal/core"
)

// MockSource is a controllable implementation of core.Source for use in tests.
type MockSource struct {
	NameVal string
	KindVal core.SourceKind
	Cfg     core.SourceConfig
	Items   []core.Item
	SyncErr error

	// SyncCalled tracks how many times Sync was called.
	SyncCalled int
	// LastSince records the since argument from the most recent Sync call.
	LastSince *time.Time
}

func (m *MockSource) Name() string              { return m.NameVal }
func (m *MockSource) Kind() core.SourceKind     { return m.KindVal }
func (m *MockSource) Config() core.SourceConfig { return m.Cfg }

func (m *MockSource) Sync(_ context.Context, since *time.Time) ([]core.Item, error) {
	m.SyncCalled++
	m.LastSince = since
	if m.SyncErr != nil {
		return nil, m.SyncErr
	}
	return m.Items, nil
}

// MockSourceConfig is a controllable implementation of core.SourceConfig.
type MockSourceConfig struct {
	ValidationErr error
}

func (m *MockSourceConfig) Validate() error { return m.ValidationErr }
