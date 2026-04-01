// Package actions is the shared application service layer for mission-control.
// Both the CLI and HTTP API call into this package; neither duplicates logic here.
package actions

import (
	"errors"
	"sync"

	"github.com/theakshaypant/mission-control/internal/core"
	syncp "github.com/theakshaypant/mission-control/internal/sync"
)

// ErrNotFound is returned when the requested item does not exist in the store.
var ErrNotFound = errors.New("item not found")

// Actions encodes all business rules for querying and mutating
// mission-control state. Construct one via New and share it across
// the CLI commands and HTTP handlers.
type Actions struct {
	store  core.Store
	mu     sync.RWMutex
	runner *syncp.Runner
}

// New returns an Actions service backed by the given store and runner.
func New(store core.Store, runner *syncp.Runner) *Actions {
	return &Actions{store: store, runner: runner}
}

// SetRunner swaps in a new runner. Safe to call concurrently with any action.
func (a *Actions) SetRunner(r *syncp.Runner) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.runner = r
}

func (a *Actions) getRunner() *syncp.Runner {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.runner
}
